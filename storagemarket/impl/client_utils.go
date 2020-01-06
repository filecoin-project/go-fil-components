package storageimpl

import (
	"context"
	"github.com/filecoin-project/go-fil-components/pieceio"
	"github.com/filecoin-project/go-fil-components/pieceio/cario"
	blocks "github.com/ipfs/go-block-format"
	"runtime"

	ipldfmt "github.com/ipfs/go-ipld-format"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-components/filestore"
	"github.com/filecoin-project/go-fil-components/pieceio/padreader"
	"github.com/filecoin-project/go-fil-components/pieceio/sectorcalculator"
	"github.com/filecoin-project/go-statestore"
)

func (c *Client) failDeal(id cid.Cid, cerr error) {
	if cerr == nil {
		_, f, l, _ := runtime.Caller(1)
		cerr = xerrors.Errorf("unknown error (fail called at %s:%d)", f, l)
	}

	s, ok := c.conns[id]
	if ok {
		_ = s.Reset()
		delete(c.conns, id)
	}

	// TODO: store in some sort of audit log
	log.Errorf("deal %s failed: %+v", id, cerr)
}

type readStoreAdapter struct {
	nodeGetter ipldfmt.NodeGetter
}

func (adapter readStoreAdapter) Get(id cid.Cid) (blocks.Block, error) {
	return adapter.nodeGetter.Get(context.Background(), id)
}

func (c *Client) commP(ctx context.Context, data cid.Cid) ([]byte, uint64, error) {
	root, err := c.dag.Get(ctx, data)
	if err != nil {
		log.Errorf("failed to get file root for deal: %s", err)
		return nil, 0, err
	}

	pr := padreader.NewPadReader()
	carIO := cario.NewCarIO()
	sectorCalculator := sectorcalculator.NewSectorCalculator("")
	fs, err := filestore.NewLocalFileStore("")
	if err != nil {
		return nil, 0, err
	}
	pio := pieceio.NewPieceIO(pr, carIO, sectorCalculator, fs)
	adapter := new(readStoreAdapter)
	adapter.nodeGetter = c.dag
	commp, tmpFile, err := pio.GeneratePieceCommitment(adapter, data, root)
	if err != nil {
		return nil, 0, xerrors.Errorf("generating CommP: %w", err)
	}
	defer func () {
		tmpFile.Close()
		fs.Delete(tmpFile.Path())
	}()
	return commp[:], uint64(tmpFile.Size()), nil
}

func (c *Client) readStorageDealResp(deal ClientDeal) (*Response, error) {
	s, ok := c.conns[deal.ProposalCid]
	if !ok {
		// TODO: Try to re-establish the connection using query protocol
		return nil, xerrors.Errorf("no connection to miner")
	}

	var resp SignedResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		log.Errorw("failed to read Response message", "error", err)
		return nil, err
	}

	if err := resp.Verify(deal.MinerWorker); err != nil {
		return nil, xerrors.Errorf("verifying response signature failed", err)
	}

	if resp.Response.Proposal != deal.ProposalCid {
		return nil, xerrors.Errorf("miner responded to a wrong proposal: %s != %s", resp.Response.Proposal, deal.ProposalCid)
	}

	return &resp.Response, nil
}

func (c *Client) disconnect(deal ClientDeal) error {
	s, ok := c.conns[deal.ProposalCid]
	if !ok {
		return nil
	}

	err := s.Close()
	delete(c.conns, deal.ProposalCid)
	return err
}

var _ datatransfer.RequestValidator = &ClientRequestValidator{}

// ClientRequestValidator validates data transfer requests for the client
// in a storage market
type ClientRequestValidator struct {
	deals *statestore.StateStore
}

// NewClientRequestValidator returns a new client request validator for the
// given datastore
func NewClientRequestValidator(deals *statestore.StateStore) *ClientRequestValidator {
	crv := &ClientRequestValidator{
		deals: deals,
	}
	return crv
}

// ValidatePush validates a push request received from the peer that will send data
// Will always error because clients should not accept push requests from a provider
// in a storage deal (i.e. send data to client).
func (c *ClientRequestValidator) ValidatePush(
	sender peer.ID,
	voucher datatransfer.Voucher,
	baseCid cid.Cid,
	Selector ipld.Node) error {
	return ErrNoPushAccepted
}

// ValidatePull validates a pull request received from the peer that will receive data
// Will succeed only if:
// - voucher has correct type
// - voucher references an active deal
// - referenced deal matches the receiver (miner)
// - referenced deal matches the given base CID
// - referenced deal is in an acceptable state
func (c *ClientRequestValidator) ValidatePull(
	receiver peer.ID,
	voucher datatransfer.Voucher,
	baseCid cid.Cid,
	Selector ipld.Node) error {
	dealVoucher, ok := voucher.(*StorageDataTransferVoucher)
	if !ok {
		return xerrors.Errorf("voucher type %s: %w", voucher.Type(), ErrWrongVoucherType)
	}

	var deal ClientDeal
	err := c.deals.Get(dealVoucher.Proposal, &deal)
	if err != nil {
		return xerrors.Errorf("Proposal CID %s: %w", dealVoucher.Proposal.String(), ErrNoDeal)
	}
	if deal.Miner != receiver {
		return xerrors.Errorf("Deal Peer %s, Data Transfer Peer %s: %w", deal.Miner.String(), receiver.String(), ErrWrongPeer)
	}
	if !deal.PayloadCid.Equals(baseCid) {
		return xerrors.Errorf("Deal Payload CID %s, Data Transfer CID %s: %w", string(deal.Proposal.PieceRef), baseCid.String(), ErrWrongPiece)
	}
	for _, state := range DataTransferStates {
		if deal.State == state {
			return nil
		}
	}
	return xerrors.Errorf("Deal State %s: %w", deal.State, ErrInacceptableDealState)
}
