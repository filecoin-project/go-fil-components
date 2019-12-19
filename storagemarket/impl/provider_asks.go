package storageimpl

import (
	"bytes"
	"context"
	"time"

	"github.com/ipfs/go-datastore"
	inet "github.com/libp2p/go-libp2p-core/network"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-components/shared/tokenamount"
	"github.com/filecoin-project/go-fil-components/shared/types"
)

func (p *Provider) SetPrice(price tokenamount.TokenAmount, ttlsecs int64) error {
	p.askLk.Lock()
	defer p.askLk.Unlock()

	var seqno uint64
	if p.ask != nil {
		seqno = p.ask.Ask.SeqNo + 1
	}

	now := time.Now().Unix()
	ask := &types.StorageAsk{
		Price:        price,
		Timestamp:    uint64(now),
		Expiry:       uint64(now + ttlsecs),
		Miner:        p.actor,
		SeqNo:        seqno,
		MinPieceSize: p.minPieceSize,
	}

	ssa, err := p.signAsk(ask)
	if err != nil {
		return err
	}

	return p.saveAsk(ssa)
}

func (p *Provider) GetAsk(m address.Address) *types.SignedStorageAsk {
	p.askLk.Lock()
	defer p.askLk.Unlock()
	if m != p.actor {
		return nil
	}

	return p.ask
}

func (p *Provider) HandleAskStream(s inet.Stream) {
	defer s.Close()
	var ar AskRequest
	if err := cborutil.ReadCborRPC(s, &ar); err != nil {
		log.Errorf("failed to read AskRequest from incoming stream: %s", err)
		return
	}

	resp := p.processAskRequest(&ar)

	if err := cborutil.WriteCborRPC(s, resp); err != nil {
		log.Errorf("failed to write ask response: %s", err)
		return
	}
}

func (p *Provider) processAskRequest(ar *AskRequest) *AskResponse {
	return &AskResponse{
		Ask: p.GetAsk(ar.Miner),
	}
}

var bestAskKey = datastore.NewKey("latest-ask")

func (p *Provider) tryLoadAsk() error {
	p.askLk.Lock()
	defer p.askLk.Unlock()

	err := p.loadAsk()
	if err != nil {
		if xerrors.Is(err, datastore.ErrNotFound) {
			log.Warn("no previous ask found, miner will not accept deals until a price is set")
			return nil
		}
		return err
	}

	return nil
}

func (p *Provider) loadAsk() error {
	askb, err := p.ds.Get(datastore.NewKey("latest-ask"))
	if err != nil {
		return xerrors.Errorf("failed to load most recent ask from disk: %w", err)
	}

	var ssa types.SignedStorageAsk
	if err := cborutil.ReadCborRPC(bytes.NewReader(askb), &ssa); err != nil {
		return err
	}

	p.ask = &ssa
	return nil
}

func (p *Provider) signAsk(a *types.StorageAsk) (*types.SignedStorageAsk, error) {
	b, err := cborutil.Dump(a)
	if err != nil {
		return nil, err
	}

	worker, err := p.spn.GetMinerWorker(context.TODO(), p.actor)
	if err != nil {
		return nil, xerrors.Errorf("failed to get worker to sign ask: %w", err)
	}

	sig, err := p.spn.SignBytes(context.TODO(), worker, b)
	if err != nil {
		return nil, err
	}

	return &types.SignedStorageAsk{
		Ask:       a,
		Signature: sig,
	}, nil
}

func (p *Provider) saveAsk(a *types.SignedStorageAsk) error {
	b, err := cborutil.Dump(a)
	if err != nil {
		return err
	}

	if err := p.ds.Put(bestAskKey, b); err != nil {
		return err
	}

	p.ask = a
	return nil
}
