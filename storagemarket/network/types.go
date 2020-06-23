package network

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/actors/crypto"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
)

//go:generate cbor-gen-for AskRequest AskResponse Proposal Response SignedResponse DealStatusRequest DealStatusResponse

// Proposal is the data sent over the network from client to provider when proposing
// a deal
type Proposal struct {
	DealProposal *market.ClientDealProposal

	Piece *storagemarket.DataRef
}

var ProposalUndefined = Proposal{}

// Response is a response to a proposal sent over the network
type Response struct {
	State storagemarket.StorageDealStatus

	// DealProposalRejected
	Message  string
	Proposal cid.Cid

	// StorageDealProposalAccepted
	PublishMessage *cid.Cid
}

// SignedResponse is a response that is signed
type SignedResponse struct {
	Response Response

	Signature *crypto.Signature
}

var SignedResponseUndefined = SignedResponse{}

// AskRequest is a request for current ask parameters for a given miner
type AskRequest struct {
	Miner address.Address
}

var AskRequestUndefined = AskRequest{}

// AskResponse is the response sent over the network in response
// to an ask request
type AskResponse struct {
	Ask *storagemarket.SignedStorageAsk
}

var AskResponseUndefined = AskResponse{}

// DealStatusRequest sent by a client to query deal status
type DealStatusRequest struct {
	Proposal  cid.Cid
	Signature crypto.Signature
}

var DealStatusRequestUndefined = DealStatusRequest{}

// DealStatusResponse is a provider's response to DealStatusRequest
type DealStatusResponse struct {
	DealState storagemarket.ProviderDealState
	Signature crypto.Signature
}

var DealStatusResponseUndefined = DealStatusResponse{}
