package dagstore

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/dagstore/mount"
)

const lotusScheme = "lotus"
const mountURLTemplate = "%s://%s"

var _ mount.Mount = (*LotusMount)(nil)

// LotusMount is the Lotus implementation of a Sharded DAG Store Mount.
// A Filecoin Piece is treated as a Shard by this implementation.
type LotusMount struct {
	api      LotusMountAPI
	pieceCid cid.Cid
}

// This method is called when registering a mount with the DAG store registry.
// The DAG store registry receives an instance of the mount (a "template").
// When the registry needs to deserialize a mount it clones the template then
// calls Deserialize on the cloned instance, which will have a reference to the
// lotus mount API supplied here.
func NewLotusMountTemplate(api LotusMountAPI) *LotusMount {
	return &LotusMount{api: api}
}

func NewLotusMount(pieceCid cid.Cid, api LotusMountAPI) (*LotusMount, error) {
	return &LotusMount{
		pieceCid: pieceCid,
		api:      api,
	}, nil
}

func (l *LotusMount) Serialize() *url.URL {
	u := fmt.Sprintf(mountURLTemplate, lotusScheme, l.pieceCid.String())
	url, err := url.Parse(u)
	if err != nil {
		// Should never happen
		panic(xerrors.Errorf("failed to parse mount URL '%s': %w", u, err))
	}

	return url
}

func (l *LotusMount) Deserialize(u *url.URL) error {
	if u.Scheme != lotusScheme {
		return xerrors.Errorf("scheme '%s' for URL '%s' does not match required scheme '%s'", u.Scheme, u, lotusScheme)
	}

	pieceCid, err := cid.Decode(u.Host)
	if err != nil {
		return xerrors.Errorf("failed to parse PieceCid from host '%s': %w", u.Host, err)
	}

	l.pieceCid = pieceCid
	return nil
}

func (l *LotusMount) Fetch(ctx context.Context) (mount.Reader, error) {
	r, err := l.api.FetchUnsealedPiece(ctx, l.pieceCid)
	if err != nil {
		return nil, xerrors.Errorf("failed to fetch unsealed piece %s: %w", l.pieceCid, err)
	}
	return &readCloser{r}, nil
}

func (l *LotusMount) Info() mount.Info {
	return mount.Info{
		Kind:             mount.KindRemote,
		AccessSequential: true,
		AccessSeek:       false,
		AccessRandom:     false,
	}
}

func (l *LotusMount) Close() error {
	return nil
}

func (l *LotusMount) Stat(_ context.Context) (mount.Stat, error) {
	size, err := l.api.GetUnpaddedCARSize(l.pieceCid)
	if err != nil {
		return mount.Stat{}, xerrors.Errorf("failed to fetch piece size for piece %s: %w", l.pieceCid, err)
	}

	// TODO Mark false when storage deal expires.
	return mount.Stat{
		Exists: true,
		Size:   int64(size),
	}, nil
}

type readCloser struct {
	io.ReadCloser
}

var _ mount.Reader = (*readCloser)(nil)

func (r *readCloser) ReadAt(p []byte, off int64) (n int, err error) {
	panic("not implemented")
}

func (r *readCloser) Seek(offset int64, whence int) (int64, error) {
	panic("not implemented")
}
