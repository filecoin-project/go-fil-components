package storageimpl

import (
	"context"

	"github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/filestore"
	"github.com/filecoin-project/go-fil-markets/pieceio"
	"github.com/filecoin-project/go-fil-markets/pieceio/cario"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket/discovery"
	"github.com/filecoin-project/go-fil-markets/shared/tokenamount"
	"github.com/filecoin-project/go-fil-markets/shared/types"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-statestore"
)

//go:generate cbor-gen-for ClientDeal ClientDealProposal

var log = logging.Logger("storagemarket_impl")

type ClientDeal struct {
	storagemarket.ClientDeal

	s network.StorageDealStream
}

type Client struct {
	net network.StorageMarketNetwork

	// dataTransfer
	// TODO: once the data transfer module is complete, the
	// client will listen to events on the data transfer module
	// Because we are using only a fake DAGService
	// implementation, there's no validation or events on the client side
	dataTransfer datatransfer.Manager
	bs           blockstore.Blockstore
	fs           filestore.FileStore
	pio          pieceio.PieceIO
	discovery    *discovery.Local

	node storagemarket.StorageClientNode

	deals *statestore.StateStore
	conns map[cid.Cid]network.StorageDealStream

	incoming chan *ClientDeal
	updated  chan clientDealUpdate

	stop    chan struct{}
	stopped chan struct{}
}

type clientDealUpdate struct {
	newState storagemarket.StorageDealStatus
	id       cid.Cid
	err      error
	mut      func(*ClientDeal)
}

func NewClient(
	net network.StorageMarketNetwork,
	bs blockstore.Blockstore,
	dataTransfer datatransfer.Manager,
	discovery *discovery.Local,
	deals *statestore.StateStore,
	scn storagemarket.StorageClientNode,
) *Client {
	carIO := cario.NewCarIO()
	pio := pieceio.NewPieceIO(carIO, bs)

	c := &Client{
		net:          net,
		dataTransfer: dataTransfer,
		bs:           bs,
		pio:          pio,
		discovery:    discovery,
		node:         scn,

		deals: deals,
		conns: map[cid.Cid]network.StorageDealStream{},

		incoming: make(chan *ClientDeal, 16),
		updated:  make(chan clientDealUpdate, 16),

		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}

	return c
}

func (c *Client) Run(ctx context.Context) {
	go func() {
		defer close(c.stopped)

		for {
			select {
			case deal := <-c.incoming:
				c.onIncoming(deal)
			case update := <-c.updated:
				c.onUpdated(ctx, update)
			case <-c.stop:
				return
			}
		}
	}()
}

func (c *Client) onIncoming(deal *ClientDeal) {
	log.Info("incoming deal")

	if _, ok := c.conns[deal.ProposalCid]; ok {
		log.Errorf("tracking deal connection: already tracking connection for deal %s", deal.ProposalCid)
		return
	}
	c.conns[deal.ProposalCid] = deal.s

	if err := c.deals.Begin(deal.ProposalCid, deal); err != nil {
		// We may have re-sent the proposal
		log.Errorf("deal tracking failed: %s", err)
		c.failDeal(deal.ProposalCid, err)
		return
	}

	go func() {
		c.updated <- clientDealUpdate{
			newState: storagemarket.StorageDealUnknown,
			id:       deal.ProposalCid,
			err:      nil,
		}
	}()
}

func (c *Client) onUpdated(ctx context.Context, update clientDealUpdate) {
	log.Infof("Client deal %s updated state to %s", update.id, storagemarket.DealStates[update.newState])
	var deal ClientDeal
	err := c.deals.Get(update.id).Mutate(func(d *ClientDeal) error {
		d.State = update.newState
		if update.mut != nil {
			update.mut(d)
		}
		deal = *d
		return nil
	})
	if update.err != nil {
		log.Errorf("deal %s failed: %s", update.id, update.err)
		c.failDeal(update.id, update.err)
		return
	}
	if err != nil {
		c.failDeal(update.id, err)
		return
	}

	switch update.newState {
	case storagemarket.StorageDealUnknown: // new
		c.handle(ctx, deal, c.new, storagemarket.StorageDealProposalAccepted)
	case storagemarket.StorageDealProposalAccepted:
		c.handle(ctx, deal, c.accepted, storagemarket.StorageDealStaged)
	case storagemarket.StorageDealStaged:
		c.handle(ctx, deal, c.staged, storagemarket.StorageDealSealing)
	case storagemarket.StorageDealSealing:
		c.handle(ctx, deal, c.sealing, storagemarket.StorageDealNoUpdate)
		// TODO: StorageDealActive -> watch for faults, expiration, etc.
	}
}

type ClientDealProposal struct {
	Data cid.Cid

	PricePerEpoch      tokenamount.TokenAmount
	ProposalExpiration uint64
	Duration           uint64

	ProviderAddress address.Address
	Client          address.Address
	MinerWorker     address.Address
	MinerID         peer.ID
}

func (c *Client) Start(ctx context.Context, p ClientDealProposal) (cid.Cid, error) {
	amount := tokenamount.Mul(p.PricePerEpoch, tokenamount.FromInt(p.Duration))
	if err := c.node.EnsureFunds(ctx, p.Client, amount); err != nil {
		return cid.Undef, xerrors.Errorf("adding market funds failed: %w", err)
	}

	commP, pieceSize, err := c.commP(ctx, p.Data)
	if err != nil {
		return cid.Undef, xerrors.Errorf("computing commP failed: %w", err)
	}

	dealProposal := &storagemarket.StorageDealProposal{
		PieceRef:             commP,
		PieceSize:            uint64(pieceSize),
		Client:               p.Client,
		Provider:             p.ProviderAddress,
		ProposalExpiration:   p.ProposalExpiration,
		Duration:             p.Duration,
		StoragePricePerEpoch: p.PricePerEpoch,
		StorageCollateral:    tokenamount.FromInt(uint64(pieceSize)), // TODO: real calc
	}

	if err := c.node.SignProposal(ctx, p.Client, dealProposal); err != nil {
		return cid.Undef, xerrors.Errorf("signing deal proposal failed: %w", err)
	}

	proposalNd, err := cborutil.AsIpld(dealProposal)
	if err != nil {
		return cid.Undef, xerrors.Errorf("getting proposal node failed: %w", err)
	}

	s, err := c.net.NewDealStream(p.MinerID)
	if err != nil {
		return cid.Undef, xerrors.Errorf("connecting to storage provider failed: %w", err)
	}

	proposal := network.Proposal{DealProposal: dealProposal, Piece: p.Data}
	if err := s.WriteDealProposal(proposal); err != nil {
		return cid.Undef, xerrors.Errorf("sending proposal to storage provider failed: %w", err)
	}

	deal := &ClientDeal{
		ClientDeal: storagemarket.ClientDeal{
			ProposalCid: proposalNd.Cid(),
			Proposal:    *dealProposal,
			State:       storagemarket.StorageDealUnknown,
			Miner:       p.MinerID,
			MinerWorker: p.MinerWorker,
			PayloadCid:  p.Data,
		},

		s: s,
	}

	c.incoming <- deal

	return deal.ProposalCid, c.discovery.AddPeer(p.Data, retrievalmarket.RetrievalPeer{
		Address: dealProposal.Provider,
		ID:      deal.Miner,
	})
}

func (c *Client) QueryAsk(ctx context.Context, p peer.ID, a address.Address) (*types.SignedStorageAsk, error) {
	s, err := c.net.NewAskStream(p)
	if err != nil {
		return nil, xerrors.Errorf("failed to open stream to miner: %w", err)
	}

	request := network.AskRequest{Miner: a}
	if err := s.WriteAskRequest(request); err != nil {
		return nil, xerrors.Errorf("failed to send ask request: %w", err)
	}

	out, err := s.ReadAskResponse()
	if err != nil {
		return nil, xerrors.Errorf("failed to read ask response: %w", err)
	}

	if out.Ask == nil {
		return nil, xerrors.Errorf("got no ask back")
	}

	if out.Ask.Ask.Miner != a {
		return nil, xerrors.Errorf("got back ask for wrong miner")
	}

	if err := c.checkAskSignature(out.Ask); err != nil {
		return nil, xerrors.Errorf("ask was not properly signed")
	}

	return out.Ask, nil
}

func (c *Client) List() ([]ClientDeal, error) {
	var out []ClientDeal
	if err := c.deals.List(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetDeal(d cid.Cid) (*ClientDeal, error) {
	var out ClientDeal
	if err := c.deals.Get(d).Get(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Stop() {
	close(c.stop)
	<-c.stopped
}
