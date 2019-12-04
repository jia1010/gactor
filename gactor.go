package gactor

import "gactor/actor"

func Call(toActorId string, params interface{}) (interface{}, error) {
	return actor.Call(toActorId, params)
}

func Cast(toActorId string, params interface{}) error {
	return actor.Cast(toActorId, params)
}

func AsyncCall(fromActorId, toActorId string, params interface{}, cb actor.RpcHandler) error {
	return actor.AsyncCall(fromActorId, toActorId, params, cb)
}
