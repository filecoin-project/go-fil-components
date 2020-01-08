package shared_testutil

import (
	"math/big"
	"math/rand"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-data-transfer/testutil"

	"github.com/filecoin-project/go-fil-components/retrievalmarket"
	"github.com/filecoin-project/go-fil-components/shared/tokenamount"
	"github.com/filecoin-project/go-fil-components/shared/types"
)

// MakeTestSignedVoucher generates a random SignedVoucher that has all non-zero fields
func MakeTestSignedVoucher() *types.SignedVoucher {
	return &types.SignedVoucher{
		TimeLock:       rand.Uint64(),
		SecretPreimage: []byte("secret-preimage"),
		Extra:          MakeTestModVerifyParams(),
		Lane:           rand.Uint64(),
		Nonce:          rand.Uint64(),
		Amount:         MakeTestTokenAmount(),
		MinCloseHeight: rand.Uint64(),
		Merges:         []types.Merge{MakeTestMerge()},
		Signature:      MakeTestSignature(),
	}
}

// MakeTestModVerifyParams generates a random ModVerifyParams that has all non-zero fields
func MakeTestModVerifyParams() *types.ModVerifyParams {
	return &types.ModVerifyParams{
		Actor:  address.TestAddress,
		Method: rand.Uint64(),
		Data:   []byte("ModVerifyParams data"),
	}
}

// MakeTestMerge generates a random Merge that has all non-zero fields
func MakeTestMerge() types.Merge {
	return types.Merge{
		Lane:  rand.Uint64(),
		Nonce: rand.Uint64(),
	}
}

// MakeTestSignagure generates a valid yet random Signature with all non-zero fields
func MakeTestSignature() *types.Signature {
	return &types.Signature{
		Type: types.KTSecp256k1,
		Data: []byte("signature data"),
	}
}

// MakeTestTokenAmount generates a valid yet random TokenAmount with a non-zero value.
func MakeTestTokenAmount() tokenamount.TokenAmount {
	return tokenamount.TokenAmount{Int: big.NewInt(rand.Int63())}
}

// MakeTestQueryResponse generates a valid, random QueryResponse with no non-zero fields
func MakeTestQueryResponse() retrievalmarket.QueryResponse {
	return retrievalmarket.QueryResponse{
		Status:                     retrievalmarket.QueryResponseUnavailable,
		Size:                       rand.Uint64(),
		PaymentAddress:             address.TestAddress2,
		MinPricePerByte:            MakeTestTokenAmount(),
		MaxPaymentInterval:         rand.Uint64(),
		MaxPaymentIntervalIncrease: rand.Uint64(),
	}
}

// MakeTestDealProposal generates a valid, random DealProposal
func MakeTestDealProposal() retrievalmarket.DealProposal {
	cid := testutil.GenerateCids(1)[0]
	return retrievalmarket.DealProposal{
		PieceCID: cid.Bytes(),
		ID:       retrievalmarket.DealID(rand.Uint64()),
		Params: retrievalmarket.Params{
			PricePerByte:            MakeTestTokenAmount(),
			PaymentInterval:         rand.Uint64(),
			PaymentIntervalIncrease: rand.Uint64(),
		},
	}
}

// MakeTestDealProposal generates a valid, random DealResponse
func MakeTestDealResponse() retrievalmarket.DealResponse {
	fakeBlk := retrievalmarket.Block{
		Prefix: []byte("prefix"),
		Data:   []byte("data"),
	}

	return retrievalmarket.DealResponse{
		Status:      retrievalmarket.DealStatusOngoing,
		ID:          retrievalmarket.DealID(rand.Uint64()),
		PaymentOwed: MakeTestTokenAmount(),
		Message:     "deal response message",
		Blocks:      []retrievalmarket.Block{fakeBlk},
	}
}
