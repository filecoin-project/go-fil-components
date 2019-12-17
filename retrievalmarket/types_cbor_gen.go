package retrievalmarket

import (
	"fmt"
	"io"

	"github.com/filecoin-project/go-fil-components/shared/types"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

var _ = xerrors.Errorf

func (t *Query) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{129}); err != nil {
		return err
	}

	// t.PieceCID ([]uint8) (slice)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajByteString, uint64(len(t.PieceCID)))); err != nil {
		return err
	}
	if _, err := w.Write(t.PieceCID); err != nil {
		return err
	}
	return nil
}

func (t *Query) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 1 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.PieceCID ([]uint8) (slice)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}

	if extra > cbg.ByteArrayMaxLen {
		return fmt.Errorf("t.PieceCID: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}
	t.PieceCID = make([]byte, extra)
	if _, err := io.ReadFull(br, t.PieceCID); err != nil {
		return err
	}
	return nil
}

func (t *QueryResponse) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{134}); err != nil {
		return err
	}

	// t.Status (retrievalmarket.QueryResponseStatus) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Status))); err != nil {
		return err
	}

	// t.Size (uint64) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Size))); err != nil {
		return err
	}

	// t.PaymentAddress (address.Address) (struct)
	if err := t.PaymentAddress.MarshalCBOR(w); err != nil {
		return err
	}

	// t.MinPricePerByte (tokenamount.TokenAmount) (struct)
	if err := t.MinPricePerByte.MarshalCBOR(w); err != nil {
		return err
	}

	// t.MaxPaymentInterval (uint64) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.MaxPaymentInterval))); err != nil {
		return err
	}

	// t.MaxPaymentIntervalIncrease (uint64) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.MaxPaymentIntervalIncrease))); err != nil {
		return err
	}
	return nil
}

func (t *QueryResponse) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 6 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Status (retrievalmarket.QueryResponseStatus) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.Status = QueryResponseStatus(extra)
	// t.Size (uint64) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.Size = uint64(extra)
	// t.PaymentAddress (address.Address) (struct)

	{

		if err := t.PaymentAddress.UnmarshalCBOR(br); err != nil {
			return err
		}

	}
	// t.MinPricePerByte (tokenamount.TokenAmount) (struct)

	{

		if err := t.MinPricePerByte.UnmarshalCBOR(br); err != nil {
			return err
		}

	}
	// t.MaxPaymentInterval (uint64) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.MaxPaymentInterval = uint64(extra)
	// t.MaxPaymentIntervalIncrease (uint64) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.MaxPaymentIntervalIncrease = uint64(extra)
	return nil
}

func (t *DealProposal) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{131}); err != nil {
		return err
	}

	// t.PieceCID ([]uint8) (slice)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajByteString, uint64(len(t.PieceCID)))); err != nil {
		return err
	}
	if _, err := w.Write(t.PieceCID); err != nil {
		return err
	}

	// t.ID (retrievalmarket.DealID) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.ID))); err != nil {
		return err
	}

	// t.Params (retrievalmarket.Params) (struct)
	if err := t.Params.MarshalCBOR(w); err != nil {
		return err
	}
	return nil
}

func (t *DealProposal) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 3 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.PieceCID ([]uint8) (slice)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}

	if extra > cbg.ByteArrayMaxLen {
		return fmt.Errorf("t.PieceCID: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}
	t.PieceCID = make([]byte, extra)
	if _, err := io.ReadFull(br, t.PieceCID); err != nil {
		return err
	}
	// t.ID (retrievalmarket.DealID) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.ID = DealID(extra)
	// t.Params (retrievalmarket.Params) (struct)

	{

		if err := t.Params.UnmarshalCBOR(br); err != nil {
			return err
		}

	}
	return nil
}

func (t *DealResponse) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{133}); err != nil {
		return err
	}

	// t.Status (retrievalmarket.DealStatus) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.Status))); err != nil {
		return err
	}

	// t.ID (retrievalmarket.DealID) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.ID))); err != nil {
		return err
	}

	// t.PaymentOwed (tokenamount.TokenAmount) (struct)
	if err := t.PaymentOwed.MarshalCBOR(w); err != nil {
		return err
	}

	// t.Message (string) (string)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajTextString, uint64(len(t.Message)))); err != nil {
		return err
	}
	if _, err := w.Write([]byte(t.Message)); err != nil {
		return err
	}

	// t.Blocks ([]retrievalmarket.Block) (slice)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajArray, uint64(len(t.Blocks)))); err != nil {
		return err
	}
	for _, v := range t.Blocks {
		if err := v.MarshalCBOR(w); err != nil {
			return err
		}
	}
	return nil
}

func (t *DealResponse) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 5 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Status (retrievalmarket.DealStatus) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.Status = DealStatus(extra)
	// t.ID (retrievalmarket.DealID) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.ID = DealID(extra)
	// t.PaymentOwed (tokenamount.TokenAmount) (struct)

	{

		if err := t.PaymentOwed.UnmarshalCBOR(br); err != nil {
			return err
		}

	}
	// t.Message (string) (string)

	{
		sval, err := cbg.ReadString(br)
		if err != nil {
			return err
		}

		t.Message = string(sval)
	}
	// t.Blocks ([]retrievalmarket.Block) (slice)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("t.Blocks: array too large (%d)", extra)
	}

	if maj != cbg.MajArray {
		return fmt.Errorf("expected cbor array")
	}
	if extra > 0 {
		t.Blocks = make([]Block, extra)
	}
	for i := 0; i < int(extra); i++ {

		var v Block
		if err := v.UnmarshalCBOR(br); err != nil {
			return err
		}

		t.Blocks[i] = v
	}

	return nil
}

func (t *Params) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{131}); err != nil {
		return err
	}

	// t.PricePerByte (tokenamount.TokenAmount) (struct)
	if err := t.PricePerByte.MarshalCBOR(w); err != nil {
		return err
	}

	// t.PaymentInterval (uint64) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.PaymentInterval))); err != nil {
		return err
	}

	// t.PaymentIntervalIncrease (uint64) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.PaymentIntervalIncrease))); err != nil {
		return err
	}
	return nil
}

func (t *Params) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 3 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.PricePerByte (tokenamount.TokenAmount) (struct)

	{

		if err := t.PricePerByte.UnmarshalCBOR(br); err != nil {
			return err
		}

	}
	// t.PaymentInterval (uint64) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.PaymentInterval = uint64(extra)
	// t.PaymentIntervalIncrease (uint64) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.PaymentIntervalIncrease = uint64(extra)
	return nil
}

func (t *QueryParams) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{128}); err != nil {
		return err
	}
	return nil
}

func (t *QueryParams) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 0 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	return nil
}

func (t *DealPayment) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{131}); err != nil {
		return err
	}

	// t.ID (retrievalmarket.DealID) (uint64)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajUnsignedInt, uint64(t.ID))); err != nil {
		return err
	}

	// t.PaymentChannel (address.Address) (struct)
	if err := t.PaymentChannel.MarshalCBOR(w); err != nil {
		return err
	}

	// t.PaymentVoucher (types.SignedVoucher) (struct)
	if err := t.PaymentVoucher.MarshalCBOR(w); err != nil {
		return err
	}
	return nil
}

func (t *DealPayment) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 3 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.ID (retrievalmarket.DealID) (uint64)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajUnsignedInt {
		return fmt.Errorf("wrong type for uint64 field")
	}
	t.ID = DealID(extra)
	// t.PaymentChannel (address.Address) (struct)

	{

		if err := t.PaymentChannel.UnmarshalCBOR(br); err != nil {
			return err
		}

	}
	// t.PaymentVoucher (types.SignedVoucher) (struct)

	{

		pb, err := br.PeekByte()
		if err != nil {
			return err
		}
		if pb == cbg.CborNull[0] {
			var nbuf [1]byte
			if _, err := br.Read(nbuf[:]); err != nil {
				return err
			}
		} else {
			t.PaymentVoucher = new(types.SignedVoucher)
			if err := t.PaymentVoucher.UnmarshalCBOR(br); err != nil {
				return err
			}
		}

	}
	return nil
}

func (t *Block) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{130}); err != nil {
		return err
	}

	// t.Prefix ([]uint8) (slice)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajByteString, uint64(len(t.Prefix)))); err != nil {
		return err
	}
	if _, err := w.Write(t.Prefix); err != nil {
		return err
	}

	// t.Data ([]uint8) (slice)
	if _, err := w.Write(cbg.CborEncodeMajorType(cbg.MajByteString, uint64(len(t.Data)))); err != nil {
		return err
	}
	if _, err := w.Write(t.Data); err != nil {
		return err
	}
	return nil
}

func (t *Block) UnmarshalCBOR(r io.Reader) error {
	br := cbg.GetPeeker(r)

	maj, extra, err := cbg.CborReadHeader(br)
	if err != nil {
		return err
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cbor input should be of type array")
	}

	if extra != 2 {
		return fmt.Errorf("cbor input had wrong number of fields")
	}

	// t.Prefix ([]uint8) (slice)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}

	if extra > cbg.ByteArrayMaxLen {
		return fmt.Errorf("t.Prefix: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}
	t.Prefix = make([]byte, extra)
	if _, err := io.ReadFull(br, t.Prefix); err != nil {
		return err
	}
	// t.Data ([]uint8) (slice)

	maj, extra, err = cbg.CborReadHeader(br)
	if err != nil {
		return err
	}

	if extra > cbg.ByteArrayMaxLen {
		return fmt.Errorf("t.Data: byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}
	t.Data = make([]byte, extra)
	if _, err := io.ReadFull(br, t.Data); err != nil {
		return err
	}
	return nil
}
