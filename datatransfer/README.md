# go-fil-components/datatransfer

A go module to perform data transfers over [ipfs/go-graphsync](https://github.com/ipfs/go-graphsync)

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)

## Table of Contents
* [Background]((https://github.com/filecoin-project/go-fil-components#background))
* [Usage]((https://github.com/filecoin-project/go-fil-components#usage))
    * [Initialize a data transfer module]((https://github.com/filecoin-project/go-fil-components#initialize-a-data-transfer-module))
    * [Register a validator](https://github.com/filecoin-project/go-fil-components#register-a-validator)
    * [Open a Push or Pull Request](https://github.com/filecoin-project/go-fil-components#open-a-push-or-pull-request)
    * [Subscribe to Events](https://github.com/filecoin-project/go-fil-components#subscribe-to-events)
* [Contribute](https://github.com/filecoin-project/go-fil-components#contribute)
* [License (Apache 2.0, MIT)](https://github.com/filecoin-project/go-fil-components#license) 

## Background

Please see the [design documentation](https://github.com/filecoin-project/go-fil-components/tree/master/datatransfer/docs/DESIGNDOC)
for this module for a high-level overview and and explanation of the terms and concepts.

## Usage

**Requires go 1.13**

Install the module in your package or app with `go get "github.com/filecoin-project/go-fil-components/datatransfer"`


### Initialize a data transfer module
1. Set up imports. You need, minimally, the following imports:
    ```go
    package mypackage

    import (
        gsimpl "github.com/ipfs/go-graphsync/impl"
        "github.com/filecoin-project/go-fil-components/datatransfer"
        "github.com/libp2p/go-libp2p-core/host"
    )
            
    ```
1. Provide or create a [libp2p host.Host](https://github.com/libp2p/go-libp2p-examples/tree/master/libp2p-host)
1. Provide or create a [go-graphsync GraphExchange](https://github.com/ipfs/go-graphsync#initializing-a-graphsync-exchange)
1. Create a new instance of GraphsyncDataTransfer
    ```go
    func NewGraphsyncDatatransfer(h host.Host, gs graphsync.GraphExchange) {
        dt := datatransfer.NewGraphSyncDataTransfer(h, gs)
    }
    ```

1. If needed, build out your voucher struct and its validator. 
    
    A push or pull request must include a voucher. The voucher's type must have been registered with 
    the node receiving the request before it's sent, otherwise the request will be rejected.  

    [datatransfer.Voucher](https://github.com/filecoin-project/go-fil-components/blob/21dd66ba370176224114b13030ee68cb785fadb2/datatransfer/types.go#L17)
    and [datatransfer.Validator](https://github.com/filecoin-project/go-fil-components/blob/21dd66ba370176224114b13030ee68cb785fadb2/datatransfer/types.go#L153)
    are the interfaces used for validation of graphsync datatransfer messages.  Voucher types plus a Validator for them must be registered
    with the peer to whom requests will be sent.  

#### Example Toy Voucher and Validator
```go
type myVoucher struct {
	data string
}

func (v *myVoucher) ToBytes() ([]byte, error) {
	return []byte(v.data), nil
}

func (v *myVoucher) FromBytes(data []byte) error {
	v.data = string(data)
	return nil
}

func (v *myVoucher) Type() string {
	return "FakeDTType"
}

type myValidator struct {
	ctx                 context.Context
	validationsReceived chan receivedValidation
}

func (vl *myValidator) ValidatePush(
	sender peer.ID,
	voucher datatransfer.Voucher,
	baseCid cid.Cid,
	selector ipld.Node) error {
    
    v := voucher.(*myVoucher)
    if v.data == "" || v.data != "validpush" {
        return errors.New("invalid")
    }   

	return nil
}

func (vl *myValidator) ValidatePull(
	receiver peer.ID,
	voucher datatransfer.Voucher,
	baseCid cid.Cid,
	selector ipld.Node) error {

    v := voucher.(*myVoucher)
    if v.data == "" || v.data != "validpull" {
        return errors.New("invalid")
    }   

	return nil
}

```


Please see 
[go-fil-components/blob/master/datatransfer/types.go](https://github.com/filecoin-project/go-fil-components/blob/master/datatransfer/types.go) 
for more detail.


### Register a validator
Before sending push or pull requests, you must register a `datatransfer.Voucher` 
by its `reflect.Type` and `dataTransfer.RequestValidator` for vouchers that
must be sent with the request.  Using the trivial examples above:
```go
    func NewGraphsyncDatatransfer(h host.Host, gs graphsync.GraphExchange) {
        dt := datatransfer.NewGraphSyncDataTransfer(h, gs)
        vouch := &myVoucher{}
        mv := &myValidator{} 
        dt.RegisterVoucherType(reflect.TypeOf(&vouch), &mv)
    }
```
    
For more detail, please see the [unit tests](https://github.com/filecoin-project/go-fil-components/blob/master/datatransfer/impl/graphsync/graphsync_impl_test.go).

### Open a Push or Pull Request
For a push or pull request you need a context, a `datatransfer.Voucher`, a host `peer.ID`, a base `cid.CID`
and a selector.
```go
    func NewGraphsyncDatatransfer(ctx context.Context, h host.Host, gs graphsync.GraphExchange) {
        dt := datatransfer.NewGraphSyncDataTransfer(h, gs)
        channelID, err := dt.OpenPullDataChannel(ctx, host2.ID(), &voucher, baseCid, selector)
    }
```

### Subscribe to Events


## Contribute
PRs are welcome!  Please first read the design docs and look over the current code.  PRs against 
master require approval of at least two maintainers.  For the rest, please see our 
[CONTRIBUTING](https://github.com/filecoin-project/go-fil-components/CONTRIBUTING.md) guide.

## License
This library is dual-licensed under Apache 2.0 and MIT terms.

Copyright 2019. Protocol Labs, Inc.