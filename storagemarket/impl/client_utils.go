package storageimpl

import (
	"context"
	"runtime"

	"github.com/filecoin-project/specs-actors/actors/abi"

	ipldfree "github.com/ipld/go-ipld-prime/impl/free"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-statestore"
)

func (c *Client) failDeal(id cid.Cid, cerr error) {
	if cerr == nil {
		_, f, l, _ := runtime.Caller(1)
		cerr = xerrors.Errorf("unknown error (fail called at %s:%d)", f, l)
	}

	s, ok := c.conns[id]
	if ok {
		_ = s.Close()
		delete(c.conns, id)
	}

	// TODO: store in some sort of audit log
	log.Errorf("deal %s failed: %+v", id, cerr)
}

func (c *Client) commP(ctx context.Context, root cid.Cid) ([]byte, abi.UnpaddedPieceSize, error) {
	ssb := builder.NewSelectorSpecBuilder(ipldfree.NodeBuilder())

	// entire DAG selector
	allSelector := ssb.ExploreRecursive(selector.RecursionLimitNone(),
		ssb.ExploreAll(ssb.ExploreRecursiveEdge())).Node()

	commp, paddedSize, err := c.pio.GeneratePieceCommitment(root, allSelector)
	if err != nil {
		return nil, 0, xerrors.Errorf("generating CommP: %w", err)
	}

	return commp[:], paddedSize, nil
}

func (c *Client) verifyResponse(resp network.SignedResponse, minerAddr address.Address) error {
	b, err := cborutil.Dump(&resp.Response)
	if err != nil {
		return err
	}
	verified := c.node.VerifySignature(*resp.Signature, minerAddr, b)
	if !verified {
		return xerrors.New("could not verify signature")
	}
	return nil
}

func (c *Client) readStorageDealResp(deal ClientDeal) (*network.Response, error) {
	s, ok := c.conns[deal.ProposalCid]
	if !ok {
		// TODO: Try to re-establish the connection using query protocol
		return nil, xerrors.Errorf("no connection to miner")
	}

	resp, err := s.ReadDealResponse()
	if err != nil {
		log.Errorw("failed to read Response message", "error", err)
		return nil, err
	}

	if err := c.verifyResponse(resp, deal.MinerWorker); err != nil {
		return nil, xerrors.Errorf("verifying response signature failed: %w", err)
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
	err := c.deals.Get(dealVoucher.Proposal).Get(&deal)
	if err != nil {
		return xerrors.Errorf("Proposal CID %s: %w", dealVoucher.Proposal.String(), ErrNoDeal)
	}

	if deal.Miner != receiver {
		return xerrors.Errorf("Deal Peer %s, Data Transfer Peer %s: %w", deal.Miner.String(), receiver.String(), ErrWrongPeer)
	}
	if !deal.DataRef.Root.Equals(baseCid) {
		return xerrors.Errorf("Deal Payload CID %s, Data Transfer CID %s: %w", deal.Proposal.PieceCID.String(), baseCid.String(), ErrWrongPiece)
	}
	for _, state := range DataTransferStates {
		if deal.State == state {
			return nil
		}
	}
	return xerrors.Errorf("Deal State %s: %w", deal.State, ErrInacceptableDealState)
}
