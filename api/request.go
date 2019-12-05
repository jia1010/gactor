/*
The MIT License (MIT)

Copyright (c) 2018 SavinMax. All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package api

import (
	"errors"
	"gactor/logger"
	"time"
)

type Request struct {
	Ctx       interface{}
	Agent     Agent       // connection agent
	ReqId     int32       // request auto incr id
	ReqType   int32       // request type
	Params    interface{} // request params
	Responsed bool        // is already responsed
	CreatedAt int64       // created at
}

type Agent interface {
	SendData(reqId int32, data []byte) error
	GetActorId() string
	Close(reason string) error
	GetUuid() string
}

var (
	ErrRouteNotFound = errors.New("route not found")
)

func NewLocalRequest(reqType int32, params interface{}) *Request {
	return NewRequest(nil, reqType, 0, params)
}

func NewRequest(agent Agent, reqType int32, reqId int32, params interface{}) *Request {
	request := &Request{
		Responsed: false,
		Agent:     agent,
		ReqId:     reqId,
		Params:    params,
		ReqType:   reqType,
		CreatedAt: time.Now().UnixNano(),
	}
	return request
}

func (req *Request) GetParams() interface{} {
	return req.Params
}

var responsedErr = errors.New("request already responsed")

func (req *Request) Response(msg interface{}) error {
	if req.Responsed {
		return responsedErr
	}

	req.Responsed = true
	used := (time.Now().UnixNano() - req.CreatedAt) / 100000

	switch req.ReqType {
	case ReqCast:
		return nil
	case ReqCall:
		logger.INFO(used, "ms RPC Response ReqId: ", req.ReqId, req.Agent.GetActorId(), " Params: ", msg)
		data, err := Encode(msg)
		if err != nil {
			return err
		}
		return req.Agent.SendData(req.ReqId, data)
	}
	return nil
}

func (req *Request) GetAgent() Agent {
	return req.Agent
}
