package network

import (
	"bufio"
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/jpillora/backoff"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
)

const maxStreamOpenAttempts = 5

var log = logging.Logger("retrieval_network")
var _ RetrievalMarketNetwork = new(libp2pRetrievalMarketNetwork)

// NewFromLibp2pHost constructs a new instance of the RetrievalMarketNetwork from a
// libp2p host
func NewFromLibp2pHost(h host.Host) RetrievalMarketNetwork {
	return &libp2pRetrievalMarketNetwork{host: h}
}

// libp2pRetrievalMarketNetwork transforms the libp2p host interface, which sends and receives
// NetMessage objects, into the graphsync network interface.
// It implements the RetrievalMarketNetwork API.
type libp2pRetrievalMarketNetwork struct {
	host host.Host
	// inbound messages from the network are forwarded to the receiver
	receiver RetrievalReceiver
}

//  NewQueryStream creates a new RetrievalQueryStream using the provided peer.ID
func (impl *libp2pRetrievalMarketNetwork) NewQueryStream(id peer.ID) (RetrievalQueryStream, error) {
	s, err := impl.openStream(context.Background(), id, retrievalmarket.QueryProtocolID)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	buffered := bufio.NewReaderSize(s, 16)
	return &queryStream{p: id, rw: s, buffered: buffered}, nil
}

func (impl *libp2pRetrievalMarketNetwork) openStream(ctx context.Context, id peer.ID, protocol protocol.ID) (network.Stream, error) {
	b := &backoff.Backoff{
		Min:    1 * time.Second,
		Max:    5 * time.Minute,
		Factor: 5,
		Jitter: true,
	}

	for {
		s, err := impl.host.NewStream(ctx, id, protocol)
		if err == nil {
			return s, err
		}

		nAttempts := b.Attempt()
		if nAttempts == maxStreamOpenAttempts {
			return nil, xerrors.Errorf("exhausted %d attempts but failed to open stream, err: %w", maxStreamOpenAttempts, err)
		}
		d := b.Duration()
		time.Sleep(d)
	}
}

// SetDelegate sets a RetrievalReceiver to handle stream data
func (impl *libp2pRetrievalMarketNetwork) SetDelegate(r RetrievalReceiver) error {
	impl.receiver = r
	impl.host.SetStreamHandler(retrievalmarket.QueryProtocolID, impl.handleNewQueryStream)
	return nil
}

// StopHandlingRequests unsets the RetrievalReceiver and would perform any other necessary
// shutdown logic.
func (impl *libp2pRetrievalMarketNetwork) StopHandlingRequests() error {
	impl.receiver = nil
	impl.host.RemoveStreamHandler(retrievalmarket.QueryProtocolID)
	return nil
}

func (impl *libp2pRetrievalMarketNetwork) handleNewQueryStream(s network.Stream) {
	if impl.receiver == nil {
		log.Warn("no receiver set")
		s.Reset() // nolint: errcheck,gosec
		return
	}
	remotePID := s.Conn().RemotePeer()
	buffered := bufio.NewReaderSize(s, 16)
	qs := &queryStream{remotePID, s, buffered}
	impl.receiver.HandleQueryStream(qs)
}

func (impl *libp2pRetrievalMarketNetwork) ID() peer.ID {
	return impl.host.ID()
}

func (impl *libp2pRetrievalMarketNetwork) AddAddrs(p peer.ID, addrs []ma.Multiaddr) {
	impl.host.Peerstore().AddAddrs(p, addrs, 8*time.Hour)
}
