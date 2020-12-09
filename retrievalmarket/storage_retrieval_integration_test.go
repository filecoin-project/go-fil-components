package retrievalmarket_test

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/actors/builtin/paych"

	"github.com/filecoin-project/go-fil-markets/piecestore"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	retrievalimpl "github.com/filecoin-project/go-fil-markets/retrievalmarket/impl"
	testnodes2 "github.com/filecoin-project/go-fil-markets/retrievalmarket/impl/testnodes"
	rmnet "github.com/filecoin-project/go-fil-markets/retrievalmarket/network"
	"github.com/filecoin-project/go-fil-markets/shared"
	"github.com/filecoin-project/go-fil-markets/shared_testutil"
	tut "github.com/filecoin-project/go-fil-markets/shared_testutil"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/testharness"
	"github.com/filecoin-project/go-fil-markets/storagemarket/testnodes"
)

func TestStorageRetrieval(t *testing.T) {
	bgCtx := context.Background()
	sh := testharness.NewHarness(t, bgCtx, true, testnodes.DelayFakeCommonNode{},
		testnodes.DelayFakeCommonNode{}, false)
	shared_testutil.StartAndWaitForReady(bgCtx, t, sh.Client)
	shared_testutil.StartAndWaitForReady(bgCtx, t, sh.Provider)

	// set up a subscriber
	providerDealChan := make(chan storagemarket.MinerDeal)
	subscriber := func(event storagemarket.ProviderEvent, deal storagemarket.MinerDeal) {
		providerDealChan <- deal
	}
	_ = sh.Provider.SubscribeToEvents(subscriber)

	clientDealChan := make(chan storagemarket.ClientDeal)
	clientSubscriber := func(event storagemarket.ClientEvent, deal storagemarket.ClientDeal) {
		clientDealChan <- deal
	}
	_ = sh.Client.SubscribeToEvents(clientSubscriber)

	// set ask price where we'll accept any price
	err := sh.Provider.SetAsk(big.NewInt(0), big.NewInt(0), 50_000)
	assert.NoError(t, err)

	result := sh.ProposeStorageDeal(t, &storagemarket.DataRef{TransferType: storagemarket.TTGraphsync, Root: sh.PayloadCid}, false, false)
	require.False(t, result.ProposalCid.Equals(cid.Undef))

	time.Sleep(time.Millisecond * 200)

	ctxTimeout, canc := context.WithTimeout(bgCtx, 25*time.Second)
	defer canc()

	var storageProviderSeenDeal storagemarket.MinerDeal
	var storageClientSeenDeal storagemarket.ClientDeal
	for storageProviderSeenDeal.State != storagemarket.StorageDealExpired ||
		storageClientSeenDeal.State != storagemarket.StorageDealExpired {
		select {
		case storageProviderSeenDeal = <-providerDealChan:
		case storageClientSeenDeal = <-clientDealChan:
		case <-ctxTimeout.Done():
			t.Fatalf("never saw completed deal, client deal state: %s (%d), provider deal state: %s (%d)",
				storagemarket.DealStates[storageClientSeenDeal.State],
				storageClientSeenDeal.State,
				storagemarket.DealStates[storageProviderSeenDeal.State],
				storageProviderSeenDeal.State,
			)
		}
	}

	rh := newRetrievalHarness(ctxTimeout, t, sh, storageClientSeenDeal)

	clientDealStateChan := make(chan retrievalmarket.ClientDealState)
	rh.Client.SubscribeToEvents(func(event retrievalmarket.ClientEvent, state retrievalmarket.ClientDealState) {
		switch event {
		case retrievalmarket.ClientEventComplete:
			clientDealStateChan <- state
		default:
			msg := `
			Client:
			Event:           %s
			Status:          %s
			TotalReceived:   %d
			BytesPaidFor:    %d
			CurrentInterval: %d
			TotalFunds:      %s
			Message:         %s
			`
			t.Logf(msg, retrievalmarket.ClientEvents[event], retrievalmarket.DealStatuses[state.Status], state.TotalReceived, state.BytesPaidFor, state.CurrentInterval,
				state.TotalFunds.String(), state.Message)
		}
	})

	providerDealStateChan := make(chan retrievalmarket.ProviderDealState)
	rh.Provider.SubscribeToEvents(func(event retrievalmarket.ProviderEvent, state retrievalmarket.ProviderDealState) {
		switch event {
		case retrievalmarket.ProviderEventCleanupComplete:
			providerDealStateChan <- state
		default:
			msg := `
			Provider:
			Event:           %s
			Status:          %s
			TotalSent:       %d
			FundsReceived:   %s
			Message:		 %s
			CurrentInterval: %d
			`
			t.Logf(msg, retrievalmarket.ProviderEvents[event], retrievalmarket.DealStatuses[state.Status], state.TotalSent, state.FundsReceived.String(), state.Message,
				state.CurrentInterval)
		}
	})

	// **** Send the query for the Piece
	// set up retrieval params
	peers := rh.Client.FindProviders(sh.PayloadCid)
	require.Len(t, peers, 1)
	retrievalPeer := peers[0]
	require.NotNil(t, retrievalPeer.PieceCID)

	rh.ClientNode.ExpectKnownAddresses(retrievalPeer, nil)

	resp, err := rh.Client.Query(bgCtx, retrievalPeer, sh.PayloadCid, retrievalmarket.QueryParams{})
	require.NoError(t, err)
	require.Equal(t, retrievalmarket.QueryResponseAvailable, resp.Status)

	// testing V1 only
	rmParams, err := retrievalmarket.NewParamsV1(rh.RetrievalParams.PricePerByte, rh.RetrievalParams.PaymentInterval, rh.RetrievalParams.PaymentIntervalIncrease, shared.AllSelector(), nil, big.Zero())
	require.NoError(t, err)

	voucherAmts := []abi.TokenAmount{abi.NewTokenAmount(10136000), abi.NewTokenAmount(9784000)}
	proof := []byte("")
	for _, voucherAmt := range voucherAmts {
		require.NoError(t, rh.ProviderNode.ExpectVoucher(*rh.ExpPaych, rh.ExpVoucher, proof, voucherAmt, voucherAmt, nil))
	}
	// just make sure there is enough to cover the transfer
	fsize := 19000 // this is the known file size of the test file lorem.txt
	expectedTotal := big.Mul(rh.RetrievalParams.PricePerByte, abi.NewTokenAmount(int64(fsize*2)))

	// *** Retrieve the piece

	clientStoreID := sh.TestData.MultiStore1.Next()
	did, err := rh.Client.Retrieve(bgCtx, sh.PayloadCid, rmParams, expectedTotal, retrievalPeer, *rh.ExpPaych, retrievalPeer.Address, &clientStoreID)
	assert.Equal(t, did, retrievalmarket.DealID(0))
	require.NoError(t, err)

	ctxTimeout, cancel := context.WithTimeout(bgCtx, 10*time.Second)
	defer cancel()

	// verify that client subscribers will be notified of state changes
	var clientDealState retrievalmarket.ClientDealState
	select {
	case <-ctxTimeout.Done():
		t.Error("deal never completed")
		t.FailNow()
	case clientDealState = <-clientDealStateChan:
	}

	ctxTimeout, cancel = context.WithTimeout(bgCtx, 5*time.Second)
	defer cancel()
	var providerDealState retrievalmarket.ProviderDealState
	select {
	case <-ctxTimeout.Done():
		t.Error("provider never saw completed deal")
		t.FailNow()
	case providerDealState = <-providerDealStateChan:
	}

	require.Equal(t, retrievalmarket.DealStatusCompleted, providerDealState.Status)
	require.Equal(t, retrievalmarket.DealStatusCompleted, clientDealState.Status)

	rh.ClientNode.VerifyExpectations(t)
	sh.TestData.VerifyFileTransferredIntoStore(t, cidlink.Link{Cid: sh.PayloadCid}, clientStoreID, false, uint64(fsize))

}

var _ datatransfer.RequestValidator = (*fakeDTValidator)(nil)

type retrievalHarness struct {
	Ctx                         context.Context
	Epoch                       abi.ChainEpoch
	Client                      retrievalmarket.RetrievalClient
	ClientNode                  *testnodes2.TestRetrievalClientNode
	Provider                    retrievalmarket.RetrievalProvider
	ProviderNode                *testnodes2.TestRetrievalProviderNode
	PieceStore                  piecestore.PieceStore
	ExpPaych, NewLaneAddr       *address.Address
	ExpPaychAmt, ActualPaychAmt *abi.TokenAmount
	ExpVoucher, ActualVoucher   *paych.SignedVoucher
	RetrievalParams             retrievalmarket.Params
}

func newRetrievalHarness(ctx context.Context, t *testing.T, sh *testharness.StorageHarness, deal storagemarket.ClientDeal) *retrievalHarness {

	var newPaychAmt abi.TokenAmount
	paymentChannelRecorder := func(client, miner address.Address, amt abi.TokenAmount) {
		newPaychAmt = amt
	}

	var newLaneAddr address.Address
	laneRecorder := func(paymentChannel address.Address) {
		newLaneAddr = paymentChannel
	}

	var newVoucher paych.SignedVoucher
	paymentVoucherRecorder := func(v *paych.SignedVoucher) {
		newVoucher = *v
	}

	cids := tut.GenerateCids(2)
	clientPaymentChannel, err := address.NewActorAddress([]byte("a"))

	expectedVoucher := tut.MakeTestSignedVoucher()
	require.NoError(t, err)
	clientNode := testnodes2.NewTestRetrievalClientNode(testnodes2.TestRetrievalClientNodeParams{
		Lane:                   expectedVoucher.Lane,
		PayCh:                  clientPaymentChannel,
		Voucher:                expectedVoucher,
		PaymentChannelRecorder: paymentChannelRecorder,
		AllocateLaneRecorder:   laneRecorder,
		PaymentVoucherRecorder: paymentVoucherRecorder,
		CreatePaychCID:         cids[0],
		AddFundsCID:            cids[1],
		IntegrationTest:        true,
	})

	nw1 := rmnet.NewFromLibp2pHost(sh.TestData.Host1, rmnet.RetryParameters(0, 0, 0, 0))
	clientDs := namespace.Wrap(sh.TestData.Ds1, datastore.NewKey("/retrievals/client"))
	client, err := retrievalimpl.NewClient(nw1, sh.TestData.MultiStore1, sh.DTClient, clientNode, sh.PeerResolver, clientDs, sh.TestData.RetrievalStoredCounter1)
	require.NoError(t, err)
	tut.StartAndWaitForReady(ctx, t, client)
	payloadCID := deal.DataRef.Root
	providerPaymentAddr := deal.MinerWorker
	providerNode := testnodes2.NewTestRetrievalProviderNode()

	carData := sh.ProviderNode.LastOnDealCompleteBytes
	sectorID := abi.SectorNumber(100000)
	offset := abi.PaddedPieceSize(1000)
	pieceInfo := piecestore.PieceInfo{
		PieceCID: tut.GenerateCids(1)[0],
		Deals: []piecestore.DealInfo{
			{
				SectorID: sectorID,
				Offset:   offset,
				Length:   abi.UnpaddedPieceSize(uint64(len(carData))).Padded(),
			},
		},
	}
	providerNode.ExpectUnseal(sectorID, offset.Unpadded(), abi.UnpaddedPieceSize(uint64(len(carData))), carData)
	// clear out provider blockstore
	allCids, err := sh.TestData.Bs2.AllKeysChan(sh.Ctx)
	require.NoError(t, err)
	for c := range allCids {
		err = sh.TestData.Bs2.DeleteBlock(c)
		require.NoError(t, err)
	}

	nw2 := rmnet.NewFromLibp2pHost(sh.TestData.Host2, rmnet.RetryParameters(0, 0, 0, 0))
	pieceStore := tut.NewTestPieceStore()
	expectedPiece := tut.GenerateCids(1)[0]
	cidInfo := piecestore.CIDInfo{
		PieceBlockLocations: []piecestore.PieceBlockLocation{
			{
				PieceCID: expectedPiece,
			},
		},
	}
	pieceStore.ExpectCID(payloadCID, cidInfo)
	pieceStore.ExpectPiece(expectedPiece, pieceInfo)
	providerDs := namespace.Wrap(sh.TestData.Ds2, datastore.NewKey("/retrievals/provider"))
	provider, err := retrievalimpl.NewProvider(providerPaymentAddr, providerNode, nw2, pieceStore, sh.TestData.MultiStore2, sh.DTProvider, providerDs)
	require.NoError(t, err)
	tut.StartAndWaitForReady(ctx, t, provider)

	params := retrievalmarket.Params{
		PricePerByte:            abi.NewTokenAmount(1000),
		PaymentInterval:         uint64(10000),
		PaymentIntervalIncrease: uint64(1000),
		UnsealPrice:             big.Zero(),
	}

	ask := provider.GetAsk()
	ask.PaymentInterval = params.PaymentInterval
	ask.PaymentIntervalIncrease = params.PaymentIntervalIncrease
	ask.PricePerByte = params.PricePerByte
	provider.SetAsk(ask)

	return &retrievalHarness{
		Ctx:             ctx,
		Client:          client,
		ClientNode:      clientNode,
		Epoch:           sh.Epoch,
		ExpPaych:        &clientPaymentChannel,
		NewLaneAddr:     &newLaneAddr,
		ActualPaychAmt:  &newPaychAmt,
		ExpVoucher:      expectedVoucher,
		ActualVoucher:   &newVoucher,
		Provider:        provider,
		ProviderNode:    providerNode,
		PieceStore:      sh.PieceStore,
		RetrievalParams: params,
	}
}

type fakeDTValidator struct{}

func (v *fakeDTValidator) ValidatePush(sender peer.ID, voucher datatransfer.Voucher, baseCid cid.Cid, selector ipld.Node) (datatransfer.VoucherResult, error) {
	return nil, nil
}

func (v *fakeDTValidator) ValidatePull(receiver peer.ID, voucher datatransfer.Voucher, baseCid cid.Cid, selector ipld.Node) (datatransfer.VoucherResult, error) {
	return nil, nil
}
