package clientstates

import (
	"github.com/filecoin-project/go-statemachine/fsm"
	"github.com/filecoin-project/specs-actors/actors/runtime/exitcode"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/clientutils"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	smnet "github.com/filecoin-project/go-fil-markets/storagemarket/network"
)

var log = logging.Logger("storagemarket_impl")

// ClientDealEnvironment is an abstraction for interacting with
// dependencies from the storage client environment
type ClientDealEnvironment interface {
	Node() storagemarket.StorageClientNode
	DealStream(proposalCid cid.Cid) (smnet.StorageDealStream, error)
	CloseStream(proposalCid cid.Cid) error
}

// ClientStateEntryFunc is the type for all state entry functions on a storage client
type ClientStateEntryFunc func(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error

// EnsureClientFunds attempts to ensure the client has enough funds for the deal being proposed
func EnsureClientFunds(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {
	node := environment.Node()

	tok, _, err := node.GetChainHead(ctx.Context())
	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventEnsureFundsFailed, xerrors.Errorf("acquiring chain head: %w", err))
	}

	mcid, err := node.EnsureFunds(ctx.Context(), deal.Proposal.Client, deal.Proposal.Client, deal.Proposal.ClientBalanceRequirement(), tok)

	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventEnsureFundsFailed, err)
	}

	// if no message was sent, and there was no error, funds were already available
	if mcid == cid.Undef {
		return ctx.Trigger(storagemarket.ClientEventFundsEnsured)
	}

	return ctx.Trigger(storagemarket.ClientEventFundingInitiated, mcid)
}

// WaitForFunding waits for an AddFunds message to appear on the chain
func WaitForFunding(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {
	node := environment.Node()

	return node.WaitForMessage(deal.AddFundsCid, storagemarket.ChainConfidence, func(code exitcode.ExitCode, bytes []byte) error {
		if code == exitcode.Ok {
			return ctx.Trigger(storagemarket.ClientEventFundsEnsured)
		}
		return ctx.Trigger(storagemarket.ClientEventEnsureFundsFailed, xerrors.Errorf("AddFunds exit code: %w", code))
	})
}

// ProposeDeal sends the deal proposal to the provider
func ProposeDeal(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {
	s, err := environment.DealStream(deal.ProposalCid)
	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventDealStreamLookupErrored, err)
	}

	proposal := network.Proposal{DealProposal: &deal.ClientDealProposal, Piece: deal.DataRef}
	if err := s.WriteDealProposal(proposal); err != nil {
		return ctx.Trigger(storagemarket.ClientEventWriteProposalFailed, err)
	}

	return ctx.Trigger(storagemarket.ClientEventDealProposed)
}

// VerifyDealResponse reads and verifies the response from the provider to the proposed deal
func VerifyDealResponse(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {
	s, err := environment.DealStream(deal.ProposalCid)
	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventDealStreamLookupErrored, err)
	}

	resp, err := s.ReadDealResponse()
	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventReadResponseFailed, err)
	}

	tok, _, err := environment.Node().GetChainHead(ctx.Context())
	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventResponseVerificationFailed)
	}

	if err := clientutils.VerifyResponse(ctx.Context(), resp, deal.MinerWorker, tok, environment.Node().VerifySignature); err != nil {
		return ctx.Trigger(storagemarket.ClientEventResponseVerificationFailed)
	}

	if resp.Response.Proposal != deal.ProposalCid {
		return ctx.Trigger(storagemarket.ClientEventResponseDealDidNotMatch, resp.Response.Proposal, deal.ProposalCid)
	}

	if resp.Response.State != storagemarket.StorageDealProposalAccepted {
		return ctx.Trigger(storagemarket.ClientEventDealRejected, resp.Response.State, resp.Response.Message)
	}

	if err := environment.CloseStream(deal.ProposalCid); err != nil {
		return ctx.Trigger(storagemarket.ClientEventStreamCloseError, err)
	}

	return ctx.Trigger(storagemarket.ClientEventDealAccepted, resp.Response.PublishMessage)
}

// ValidateDealPublished confirms with the chain that a deal was published
func ValidateDealPublished(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {

	dealID, err := environment.Node().ValidatePublishedDeal(ctx.Context(), deal)
	if err != nil {
		return ctx.Trigger(storagemarket.ClientEventDealPublishFailed, err)
	}

	return ctx.Trigger(storagemarket.ClientEventDealPublished, dealID)
}

// VerifyDealActivated confirms that a deal was successfully committed to a sector and is active
func VerifyDealActivated(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {
	cb := func(err error) {
		if err != nil {
			_ = ctx.Trigger(storagemarket.ClientEventDealActivationFailed, err)
		} else {
			_ = ctx.Trigger(storagemarket.ClientEventDealActivated)
		}
	}

	if err := environment.Node().OnDealSectorCommitted(ctx.Context(), deal.Proposal.Provider, deal.DealID, cb); err != nil {
		return ctx.Trigger(storagemarket.ClientEventDealActivationFailed, err)
	}

	return nil
}

// FailDeal cleans up a failing deal
func FailDeal(ctx fsm.Context, environment ClientDealEnvironment, deal storagemarket.ClientDeal) error {

	if err := environment.CloseStream(deal.ProposalCid); err != nil {
		return ctx.Trigger(storagemarket.ClientEventStreamCloseError, err)
	}

	// TODO: store in some sort of audit log
	log.Errorf("deal %s failed: %s", deal.ProposalCid, deal.Message)

	return ctx.Trigger(storagemarket.ClientEventFailed)
}
