## Description
A distributed actor model implemented with golang and etcd.

## Install
```bash
# install gactor
go get github.com/mafei198/gactor

# install etcd
brew install etcd
```

## Get Started
```go
// start gactor server
err := gactor.Start()
if err != nil {
    logger.ERR("handle err: ", err)
}

// register protobuf msg factory
api.RegisterFactory(func() proto.Message {
    return &protos.Player{}
})

// register msg handler for protobuf msg
api.RegisterHandler(&protos.Player{}, func(req *api.Request) proto.Message {
    logger.INFO("handle request: ", req)
    return req.Params.(*protos.Player)
})

// request actor sync
gactor.Call(toActorId, params)

// request actor async
gactor.AsyncCall(fromActorId, toActorId, params)

// send msg async
gactor.Cast(toActorId, params)
```
* [example](example)
