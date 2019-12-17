module github.com/filecoin-project/go-fil-components

go 1.13

require (
	github.com/filecoin-project/filecoin-ffi v0.0.0-20191210104338-2383ce072e95
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/hannahhoward/cbor-gen-for v0.0.0-20191216214420-3e450425c40c
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-blockservice v0.1.3-0.20190908200855-f22eea50656c
	github.com/ipfs/go-car v0.0.3-0.20191203022317-23b0a85fd1b1
	github.com/ipfs/go-cid v0.0.4-0.20191112011718-79e75dffeb10
	github.com/ipfs/go-datastore v0.1.0
	github.com/ipfs/go-graphsync v0.0.4
	github.com/ipfs/go-ipfs-blockstore v0.1.0
	github.com/ipfs/go-ipfs-blocksutil v0.0.1
	github.com/ipfs/go-ipfs-chunker v0.0.1
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipfs-files v0.0.4
	github.com/ipfs/go-ipld-cbor v0.0.3
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v1.0.0
	github.com/ipfs/go-merkledag v0.2.4
	github.com/ipfs/go-unixfs v0.2.2-0.20190827150610-868af2e9e5cb
	github.com/ipld/go-ipld-prime v0.0.2-0.20191108012745-28a82f04c785
	github.com/ipsn/go-secp256k1 v0.0.0-20180726113642-9d62b9f0bc52
	github.com/jbenet/go-random v0.0.0-20190219211222-123a90aedc0c
	github.com/libp2p/go-libp2p v0.3.0
	github.com/libp2p/go-libp2p-blankhost v0.1.4 // indirect
	github.com/libp2p/go-libp2p-core v0.2.4
	github.com/libp2p/go-libp2p-record v0.1.1 // indirect
	github.com/libp2p/go-libp2p-swarm v0.2.2 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/multiformats/go-multiaddr-dns v0.2.0 // indirect
	github.com/multiformats/go-varint v0.0.1
	github.com/polydawn/refmt v0.0.0-20190809202753-05966cbd336a
	github.com/stretchr/testify v1.4.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20191212224538-d370462a7e8a
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
