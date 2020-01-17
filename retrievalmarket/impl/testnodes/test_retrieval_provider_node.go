package testnodes

import (
	"context"
	"errors"
	"testing"

	"github.com/filecoin-project/go-address"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/shared/tokenamount"
	"github.com/filecoin-project/go-fil-markets/shared/types"
)

type TestRetrievalProviderNodeParams struct {
	/*	PayCh        address.Address
		PayChErr     error
		Lane         uint64
		LaneError    error
		Voucher      *types.SignedVoucher
		VoucherError error*/
}

type expectedVoucherKey struct {
	paymentChannel string
	voucher        string
	proof          string
	expectedAmount string
}

type voucherResult struct {
	amount tokenamount.TokenAmount
	err    error
}

type TestRetrievalProviderNode struct {
	bs                    blockstore.Blockstore
	expectedPieces        map[string]uint64
	expectedMissingPieces map[string]struct{}
	receivedPiecesSizes   map[string]struct{}
	receivedMissingPieces map[string]struct{}
	expectedVouchers      map[expectedVoucherKey]voucherResult
	receivedVouchers      map[expectedVoucherKey]struct{}
}

var _ retrievalmarket.RetrievalProviderNode = &TestRetrievalProviderNode{}

func NewTestRetrievalProviderNode() *TestRetrievalProviderNode {
	return &TestRetrievalProviderNode{
		expectedPieces:        make(map[string]uint64),
		expectedMissingPieces: make(map[string]struct{}),
		receivedPiecesSizes:   make(map[string]struct{}),
		receivedMissingPieces: make(map[string]struct{}),
		expectedVouchers:      make(map[expectedVoucherKey]voucherResult),
		receivedVouchers:      make(map[expectedVoucherKey]struct{}),
	}
}

func (trpn *TestRetrievalProviderNode) ExpectPiece(pieceCid []byte, size uint64) {
	trpn.expectedPieces[string(pieceCid)] = size
}

func (trpn *TestRetrievalProviderNode) ExpectMissingPiece(pieceCid []byte) {
	trpn.expectedMissingPieces[string(pieceCid)] = struct{}{}
}

func (trpn *TestRetrievalProviderNode) VerifyExpectations(t *testing.T) {
	require.Equal(t, len(trpn.expectedPieces), len(trpn.receivedPiecesSizes))
	require.Equal(t, len(trpn.expectedMissingPieces), len(trpn.receivedMissingPieces))
	require.Equal(t, len(trpn.expectedVouchers), len(trpn.receivedVouchers))
}

func (trpn *TestRetrievalProviderNode) GetPieceSize(pieceCid []byte) (uint64, error) {
	size, ok := trpn.expectedPieces[string(pieceCid)]
	if ok {
		trpn.receivedPiecesSizes[string(pieceCid)] = struct{}{}
		return size, nil
	}
	_, ok = trpn.expectedMissingPieces[string(pieceCid)]
	if ok {
		trpn.receivedMissingPieces[string(pieceCid)] = struct{}{}
		return 0, retrievalmarket.ErrNotFound
	}
	return 0, errors.New("GetPieceSize failed")
}

func (trpn *TestRetrievalProviderNode) SetBlockstore(bs blockstore.Blockstore) {
	trpn.bs = bs
}

func (trpn *TestRetrievalProviderNode) SealedBlockstore(approveUnseal func() error) blockstore.Blockstore {
	return trpn.bs
}

func (trpn *TestRetrievalProviderNode) SavePaymentVoucher(
	ctx context.Context,
	paymentChannel address.Address,
	voucher *types.SignedVoucher,
	proof []byte,
	expectedAmount tokenamount.TokenAmount) (tokenamount.TokenAmount, error) {
	key, err := trpn.toExpectedVoucherKey(paymentChannel, voucher, proof, expectedAmount)
	if err != nil {
		return tokenamount.Empty, err
	}
	result, ok := trpn.expectedVouchers[key]
	if ok {
		trpn.receivedVouchers[key] = struct{}{}
		return result.amount, result.err
	}
	return tokenamount.Empty, errors.New("SavePaymentVoucher failed")
}

// --- Non-interface Functions

// to ExpectedVoucherKey creates a lookup key for expected vouchers.
func (trpn *TestRetrievalProviderNode) toExpectedVoucherKey(paymentChannel address.Address, voucher *types.SignedVoucher, proof []byte, expectedAmount tokenamount.TokenAmount) (expectedVoucherKey, error) {
	pcString := paymentChannel.String()
	voucherString, err := voucher.EncodedString()
	if err != nil {
		return expectedVoucherKey{}, err
	}
	proofString := string(proof)
	expectedAmountString := expectedAmount.String()
	return expectedVoucherKey{pcString, voucherString, proofString, expectedAmountString}, nil
}

// ExpectVoucher sets a voucher to be expected by SavePaymentVoucher
//     paymentChannel: the address of the payment channel the client creates
//     voucher: the voucher to match
//     proof: the proof to use (can be blank)
// 	   expectedAmount: the expected tokenamount for this voucher
//     actualAmount: the actual amount to use.  use same as expectedAmount unless you want to trigger an error
//     expectedErr:  an error message to expect
func (trpn *TestRetrievalProviderNode) ExpectVoucher(
	paymentChannel address.Address,
	voucher *types.SignedVoucher,
	proof []byte,
	expectedAmount tokenamount.TokenAmount,
	actualAmount tokenamount.TokenAmount,   // the actual amount it should have (same unless you want to trigger an error)
	expectedErr error) error {
	key, err := trpn.toExpectedVoucherKey(paymentChannel, voucher, proof, expectedAmount)
	if err != nil {
		return err
	}
	trpn.expectedVouchers[key] = voucherResult{actualAmount, expectedErr}
	return nil
}