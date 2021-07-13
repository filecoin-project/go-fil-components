package dagstore

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	blocksutil "github.com/ipfs/go-ipfs-blocksutil"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/dagstore/mount"

	mock_dagstore "github.com/filecoin-project/go-fil-markets/dagstore/mocks"
)

func TestLotusMount(t *testing.T) {
	ctx := context.Background()
	bgen := blocksutil.NewBlockGenerator()
	cid := bgen.Next().Cid()

	mockCtrl := gomock.NewController(t)
	// when test is done, assert expectations on all mock objects.
	defer mockCtrl.Finish()

	// create a mock lotus api that returns the reader we want
	mockLotusMountAPI := mock_dagstore.NewMockLotusMountAPI(mockCtrl)
	mockLotusMountAPI.EXPECT().FetchUnsealedPiece(gomock.Any(), cid).Return(&readCloser{ioutil.NopCloser(strings.NewReader("testing"))}, nil).Times(1)
	mockLotusMountAPI.EXPECT().FetchUnsealedPiece(gomock.Any(), cid).Return(&readCloser{ioutil.NopCloser(strings.NewReader("testing"))}, nil).Times(1)
	mockLotusMountAPI.EXPECT().GetUnpaddedCARSize(cid).Return(uint64(100), nil).Times(1)

	mnt, err := NewLotusMount(cid, mockLotusMountAPI)
	require.NoError(t, err)
	info := mnt.Info()
	require.Equal(t, info.Kind, mount.KindRemote)

	// fetch and assert success
	rd, err := mnt.Fetch(context.Background())
	require.NoError(t, err)

	bz, err := ioutil.ReadAll(rd)
	require.NoError(t, err)
	require.NoError(t, rd.Close())
	require.Equal(t, []byte("testing"), bz)

	stat, err := mnt.Stat(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 100, stat.Size)

	// serialize url then deserialize from mount template -> should get back
	// the same mount
	url := mnt.Serialize()
	mnt2 := NewLotusMountTemplate(mockLotusMountAPI)
	err = mnt2.Deserialize(url)
	require.NoError(t, err)

	// fetching on this mount should get us back the same data.
	rd, err = mnt2.Fetch(context.Background())
	require.NoError(t, err)
	bz, err = ioutil.ReadAll(rd)
	require.NoError(t, err)
	require.NoError(t, rd.Close())
	require.Equal(t, []byte("testing"), bz)
}

func TestLotusMountDeserialize(t *testing.T) {
	api := &lotusMountApiImpl{}

	bgen := blocksutil.NewBlockGenerator()
	cid := bgen.Next().Cid()

	// success
	us := fmt.Sprintf(mountURLTemplate, lotusScheme, cid.String())
	u, err := url.Parse(us)
	require.NoError(t, err)

	mnt := NewLotusMountTemplate(api)
	err = mnt.Deserialize(u)
	require.NoError(t, err)

	require.Equal(t, cid, mnt.pieceCid)
	require.Equal(t, api, mnt.api)

	// fails if scheme is not Lotus
	us = fmt.Sprintf(mountURLTemplate, "http", cid.String())
	u, err = url.Parse(us)
	require.NoError(t, err)

	err = mnt.Deserialize(u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")

	// fails if cid is not valid
	us = fmt.Sprintf(mountURLTemplate, lotusScheme, "rand")
	u, err = url.Parse(us)
	require.NoError(t, err)
	err = mnt.Deserialize(u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse PieceCid")
}
