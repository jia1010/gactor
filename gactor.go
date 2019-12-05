package gactor

import "gactor/actor"

func Call(toActorId string, params interface{}) (interface{}, error) {
	return actor.Call(toActorId, params)
}

func Cast(toActorId string, params interface{}) error {
	return actor.Cast(toActorId, params)
}

func RpcCall(toActorId string, params interface{}) (interface{}, error) {
	return actor.RpcCall(toActorId, params)
}

func RpcCast(toActorId string, params interface{}) error {
	return actor.RpcCast(toActorId, params)
}

func RpcAsyncCall(fromActorId, toActorId string, params interface{}, cb actor.RpcHandler) error {
	return actor.RpcAsyncCall(fromActorId, toActorId, params, cb)
}
