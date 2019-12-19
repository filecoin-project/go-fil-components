package storageimpl_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	dss "github.com/ipfs/go-datastore/sync"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	"github.com/libp2p/go-libp2p-core/peer"
	xerrors "golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-components/shared/types"
	"github.com/filecoin-project/go-fil-components/storagemarket"
	deals "github.com/filecoin-project/go-fil-components/storagemarket/impl"
	"github.com/filecoin-project/go-statestore"
)

var blockGenerator = blocksutil.NewBlockGenerator()

type wrongDTType struct {
}

func (wrongDTType) ToBytes() ([]byte, error) {
	return []byte{}, nil
}

func (wrongDTType) FromBytes([]byte) error {
	return fmt.Errorf("not implemented")
}

func (wrongDTType) Type() string {
	return "WrongDTTYPE"
}

func uniqueStorageDealProposal() (storagemarket.StorageDealProposal, error) {
	clientAddr, err := address.NewIDAddress(uint64(rand.Int()))
	if err != nil {
		return storagemarket.StorageDealProposal{}, err
	}
	providerAddr, err := address.NewIDAddress(uint64(rand.Int()))
	if err != nil {
		return storagemarket.StorageDealProposal{}, err
	}
	return storagemarket.StorageDealProposal{
		PieceRef: blockGenerator.Next().Cid().Bytes(),
		Client:   clientAddr,
		Provider: providerAddr,
		ProposerSignature: &types.Signature{
			Data: []byte("foo bar cat dog"),
			Type: types.KTBLS,
		},
	}, nil
}

func newClientDeal(minerID peer.ID, state storagemarket.DealState) (deals.ClientDeal, error) {
	newProposal, err := uniqueStorageDealProposal()
	if err != nil {
		return deals.ClientDeal{}, err
	}
	proposalNd, err := cborutil.AsIpld(&newProposal)
	if err != nil {
		return deals.ClientDeal{}, err
	}
	minerAddr, err := address.NewIDAddress(uint64(rand.Int()))
	if err != nil {
		return deals.ClientDeal{}, err
	}

	return deals.ClientDeal{
		ClientDeal: storagemarket.ClientDeal{
			Proposal:    newProposal,
			ProposalCid: proposalNd.Cid(),
			PayloadCid:  blockGenerator.Next().Cid(),
			Miner:       minerID,
			MinerWorker: minerAddr,
			State:       state,
		},
	}, nil
}

func newMinerDeal(clientID peer.ID, state storagemarket.DealState) (deals.MinerDeal, error) {
	newProposal, err := uniqueStorageDealProposal()
	if err != nil {
		return deals.MinerDeal{}, err
	}
	proposalNd, err := cborutil.AsIpld(&newProposal)
	if err != nil {
		return deals.MinerDeal{}, err
	}
	ref := blockGenerator.Next().Cid()

	return deals.MinerDeal{
		MinerDeal: storagemarket.MinerDeal{
			Proposal:    newProposal,
			ProposalCid: proposalNd.Cid(),
			Client:      clientID,
			State:       state,
			Ref:         ref,
		},
	}, nil
}

func TestClientRequestValidation(t *testing.T) {
	ds := dss.MutexWrap(datastore.NewMapDatastore())
	state := statestore.New(namespace.Wrap(ds, datastore.NewKey("/deals/client")))

	crv := deals.NewClientRequestValidator(state)
	minerID := peer.ID("fakepeerid")
	block := blockGenerator.Next()
	t.Run("ValidatePush fails", func(t *testing.T) {
		if !xerrors.Is(crv.ValidatePush(minerID, wrongDTType{}, block.Cid(), nil), deals.ErrNoPushAccepted) {
			t.Fatal("Push should fail for the client request validator for storage deals")
		}
	})
	t.Run("ValidatePull fails deal not found", func(t *testing.T) {
		proposal, err := uniqueStorageDealProposal()
		if err != nil {
			t.Fatal("error creating proposal")
		}
		proposalNd, err := cborutil.AsIpld(&proposal)
		if err != nil {
			t.Fatal("error serializing proposal")
		}
		pieceRef, err := cid.Cast(proposal.PieceRef)
		if err != nil {
			t.Fatal("unable to construct piece cid")
		}
		if !xerrors.Is(crv.ValidatePull(minerID, &deals.StorageDataTransferVoucher{proposalNd.Cid(), 1}, pieceRef, nil), deals.ErrNoDeal) {
			t.Fatal("Pull should fail if there is no deal stored")
		}
	})
	t.Run("ValidatePull fails wrong client", func(t *testing.T) {
		otherMiner := peer.ID("otherminer")
		clientDeal, err := newClientDeal(otherMiner, storagemarket.DealAccepted)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(clientDeal.ProposalCid, &clientDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		payloadCid := clientDeal.PayloadCid
		if !xerrors.Is(crv.ValidatePull(minerID, &deals.StorageDataTransferVoucher{clientDeal.ProposalCid, 1}, payloadCid, nil), deals.ErrWrongPeer) {
			t.Fatal("Pull should fail if miner address is incorrect")
		}
	})
	t.Run("ValidatePull fails wrong piece ref", func(t *testing.T) {
		clientDeal, err := newClientDeal(minerID, storagemarket.DealAccepted)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(clientDeal.ProposalCid, &clientDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		if !xerrors.Is(crv.ValidatePull(minerID, &deals.StorageDataTransferVoucher{clientDeal.ProposalCid, 1}, blockGenerator.Next().Cid(), nil), deals.ErrWrongPiece) {
			t.Fatal("Pull should fail if piece ref is incorrect")
		}
	})
	t.Run("ValidatePull fails wrong deal state", func(t *testing.T) {
		clientDeal, err := newClientDeal(minerID, storagemarket.DealComplete)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(clientDeal.ProposalCid, &clientDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		payloadCid := clientDeal.PayloadCid
		if !xerrors.Is(crv.ValidatePull(minerID, &deals.StorageDataTransferVoucher{clientDeal.ProposalCid, 1}, payloadCid, nil), deals.ErrInacceptableDealState) {
			t.Fatal("Pull should fail if deal is in a state that cannot be data transferred")
		}
	})
	t.Run("ValidatePull succeeds", func(t *testing.T) {
		clientDeal, err := newClientDeal(minerID, storagemarket.DealAccepted)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(clientDeal.ProposalCid, &clientDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		payloadCid := clientDeal.PayloadCid
		if crv.ValidatePull(minerID, &deals.StorageDataTransferVoucher{clientDeal.ProposalCid, 1}, payloadCid, nil) != nil {
			t.Fatal("Pull should should succeed when all parameters are correct")
		}
	})
}

func TestProviderRequestValidation(t *testing.T) {
	ds := dss.MutexWrap(datastore.NewMapDatastore())
	state := statestore.New(namespace.Wrap(ds, datastore.NewKey("/deals/client")))

	mrv := deals.NewProviderRequestValidator(state)
	clientID := peer.ID("fakepeerid")
	block := blockGenerator.Next()
	t.Run("ValidatePull fails", func(t *testing.T) {
		if !xerrors.Is(mrv.ValidatePull(clientID, wrongDTType{}, block.Cid(), nil), deals.ErrNoPullAccepted) {
			t.Fatal("Pull should fail for the provider request validator for storage deals")
		}
	})

	t.Run("ValidatePush fails deal not found", func(t *testing.T) {
		proposal, err := uniqueStorageDealProposal()
		if err != nil {
			t.Fatal("error creating proposal")
		}
		proposalNd, err := cborutil.AsIpld(&proposal)
		if err != nil {
			t.Fatal("error serializing proposal")
		}
		pieceRef, err := cid.Cast(proposal.PieceRef)
		if err != nil {
			t.Fatal("unable to construct piece cid")
		}
		if !xerrors.Is(mrv.ValidatePush(clientID, &deals.StorageDataTransferVoucher{proposalNd.Cid(), 1}, pieceRef, nil), deals.ErrNoDeal) {
			t.Fatal("Push should fail if there is no deal stored")
		}
	})
	t.Run("ValidatePush fails wrong miner", func(t *testing.T) {
		otherClient := peer.ID("otherclient")
		minerDeal, err := newMinerDeal(otherClient, storagemarket.DealAccepted)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(minerDeal.ProposalCid, &minerDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		ref := minerDeal.Ref
		if !xerrors.Is(mrv.ValidatePush(clientID, &deals.StorageDataTransferVoucher{minerDeal.ProposalCid, 1}, ref, nil), deals.ErrWrongPeer) {
			t.Fatal("Push should fail if miner address is incorrect")
		}
	})
	t.Run("ValidatePush fails wrong piece ref", func(t *testing.T) {
		minerDeal, err := newMinerDeal(clientID, storagemarket.DealAccepted)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(minerDeal.ProposalCid, &minerDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		if !xerrors.Is(mrv.ValidatePush(clientID, &deals.StorageDataTransferVoucher{minerDeal.ProposalCid, 1}, blockGenerator.Next().Cid(), nil), deals.ErrWrongPiece) {
			t.Fatal("Push should fail if piece ref is incorrect")
		}
	})
	t.Run("ValidatePush fails wrong deal state", func(t *testing.T) {
		minerDeal, err := newMinerDeal(clientID, storagemarket.DealComplete)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(minerDeal.ProposalCid, &minerDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		ref := minerDeal.Ref
		if !xerrors.Is(mrv.ValidatePush(clientID, &deals.StorageDataTransferVoucher{minerDeal.ProposalCid, 1}, ref, nil), deals.ErrInacceptableDealState) {
			t.Fatal("Push should fail if deal is in a state that cannot be data transferred")
		}
	})
	t.Run("ValidatePush succeeds", func(t *testing.T) {
		minerDeal, err := newMinerDeal(clientID, storagemarket.DealAccepted)
		if err != nil {
			t.Fatal("error creating client deal")
		}
		if err := state.Begin(minerDeal.ProposalCid, &minerDeal); err != nil {
			t.Fatal("deal tracking failed")
		}
		ref := minerDeal.Ref
		if mrv.ValidatePush(clientID, &deals.StorageDataTransferVoucher{minerDeal.ProposalCid, 1}, ref, nil) != nil {
			t.Fatal("Push should should succeed when all parameters are correct")
		}
	})
}
