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

## License
GOS is under The MIT License (MIT)

Copyright (c) 2018-2028
Savin Max <mafei.198@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

