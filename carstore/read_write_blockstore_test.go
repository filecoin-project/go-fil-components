package carstore_test

import (
	"context"
	"path/filepath"
	"testing"

	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-fil-markets/carstore"
	tut "github.com/filecoin-project/go-fil-markets/shared_testutil"
)

func TestReadWriteStoreTracker(t *testing.T) {
	ctx := context.Background()

	// Create a CARv2 file from a fixture
	testData := tut.NewLibp2pTestData(ctx, t)
	fpath1 := filepath.Join("retrievalmarket", "impl", "fixtures", "lorem.txt")
	lnk1, carFilePath1 := testData.LoadUnixFSFileToStore(t, fpath1)
	rootCidLnk1, ok := lnk1.(cidlink.Link)
	require.True(t, ok)
	fpath2 := filepath.Join("retrievalmarket", "impl", "fixtures", "lorem_under_1_block.txt")
	lnk2, carFilePath2 := testData.LoadUnixFSFileToStore(t, fpath2)
	rootCidLnk2, ok := lnk2.(cidlink.Link)
	require.True(t, ok)

	k1 := "k1"
	k2 := "k2"
	tracker := carstore.NewCarReadWriteStoreTracker()

	// Get a non-existent key
	_, err := tracker.Get(k1)
	require.True(t, carstore.IsNotFound(err))

	// Create a blockstore by calling GetOrCreate
	rdOnlyBS1, err := tracker.GetOrCreate(k1, carFilePath1, rootCidLnk1.Cid)
	require.NoError(t, err)

	// Get the blockstore using its key
	got, err := tracker.Get(k1)
	require.NoError(t, err)

	// Verify the blockstore is the same
	len1 := getBstoreLen(ctx, t, rdOnlyBS1)
	lenGot := getBstoreLen(ctx, t, got)
	require.Equal(t, len1, lenGot)

	// Call GetOrCreate with a different CAR file
	rdOnlyBS2, err := tracker.GetOrCreate(k2, carFilePath2, rootCidLnk2.Cid)
	require.NoError(t, err)

	// Verify the blockstore is different
	len2 := getBstoreLen(ctx, t, rdOnlyBS2)
	require.NotEqual(t, len1, len2)

	// Clean the second blockstore from the tracker
	err = tracker.CleanBlockstore(k2)
	require.NoError(t, err)

	// Verify it's been removed
	_, err = tracker.Get(k2)
	require.True(t, carstore.IsNotFound(err))
}
