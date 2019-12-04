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
package actor

import (
	"context"
	"encoding/json"
	"errors"
	"gactor/cluster"
	"gactor/etcd"
	"gactor/logger"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/rs/xid"
	"go.etcd.io/etcd/clientv3"
	"sync"
	"time"
)

type Meta struct {
	Uuid        string           `json:"uuid"`
	Category    string           `json:"category"`
	Dispatch    *Dispatch        `json:"dispatch"`
	NodeId      string           `json:"node_id"`
	ServerId    string           `json:"server_id"`
	ActiveAt    int64            `json:"active_at"`
	CreatedAt   int64            `json:"created_at"`
	ModRevision int64            `json:"-"`
	KV          *mvccpb.KeyValue `json:"-"`
}

const (
	DispatchTypeDefault = iota // 按负载分配
	DispatchTypeRole           // 在指定节点类型范围，按负载分配
	DispatchTypeInMap          // 部署到地图服务所在节点
)

type Dispatch struct {
	Type     int
	Value    string
	IsDaemon bool
}

const (
	MetaExpireTime    = 1800 // seconds
	MetaCacheDuration = 600  // seconds
)

type CacheMeta struct {
	Meta *Meta
	Kv   *mvccpb.KeyValue
}

var CacheMetas map[string]*Meta

func FindOrCreate(category, uuid, serverId string, dispatch *Dispatch) (*Meta, error) {
	if meta := getMetaCache(uuid); meta != nil {
		return meta, nil
	}
	meta, err := GetMeta(uuid)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return AddMeta(category, uuid, serverId, dispatch)
	}
	return meta, err
}

func AddMeta(category, uuid, serverId string, dispatch *Dispatch) (*Meta, error) {
	if meta, err := GetMeta(uuid); err == nil || err != ErrActorMetaNotExists {
		return meta, err
	}
	now := time.Now().Unix()
	meta := &Meta{
		Uuid:      uuid,
		Category:  category,
		ServerId:  serverId,
		Dispatch:  dispatch,
		ActiveAt:  now,
		CreatedAt: now,
	}
	meta, err := setToEtcd(meta)
	if err != nil {
		logger.ERR("AddMeta failed: ", meta, err)
		return nil, err
	}
	meta, err = dispatchActor(meta)
	if err != nil {
		logger.ERR("dispatchActor failed: ", meta, err)
		return nil, err
	}
	return meta, nil
}

var ErrActorMetaNotExists = errors.New("actor meta not exists")

func GetMeta(uuid string) (meta *Meta, err error) {
	if meta = getMetaCache(uuid); meta == nil {
		meta, err = getFromEtcd(uuid)
		if err != nil {
			return
		}
		if meta == nil {
			return meta, ErrActorMetaNotExists
		}
	}
	if meta.NodeId == "" {
		return dispatchActor(meta)
	}
	if _, ok := cluster.FindNode(meta.NodeId); !ok {
		return dispatchActor(meta)
	}
	return meta, nil
}

func (meta *Meta) GetNode() (*cluster.Node, bool) {
	return cluster.FindNode(meta.NodeId)
}

// after actor offline
func ExpireMeta(uuid string) {
	var err error
	var meta *Meta
	if meta, err = GetMeta(uuid); err == nil {
		meta.NodeId = ""
		_, err = setToEtcd(meta)
	}
	if err != nil {
		logger.ERR("ExpireMeta failed: ", err, uuid)
	}
}

// Add daemon actor
func AddDaemonMeta(meta *Meta) error {
	if meta.Dispatch.IsDaemon && meta.NodeId != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		key := daemonKey(meta.Uuid)
		_, err := etcd.Client.Put(ctx, key, meta.Uuid)
		return err
	}
	return nil
}

// Get all daemon actors
func GetDaemonMetaIds() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rsp, err := etcd.Client.Get(ctx, daemonPrefix(), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	metaIds := make([]string, 0)
	for _, kv := range rsp.Kvs {
		metaId := string(kv.Value)
		metaIds = append(metaIds, metaId)
	}
	return metaIds, nil
}

var errNodeNotFound = errors.New("dispatch failed, node not found")

func dispatchActor(meta *Meta) (*Meta, error) {
	game, err := chooseNodeForMeta(meta)
	if err != nil {
		return nil, err
	}
	if game == nil {
		return nil, errNodeNotFound
	}
	meta.NodeId = game.Uuid
	meta, err = setToEtcd(meta)
	if err != nil {
		return nil, err
	}
	return meta, AddDaemonMeta(meta)
}

func chooseNodeForMeta(meta *Meta) (*cluster.Node, error) {
	dispatch := meta.Dispatch
	var game *cluster.Node
	var err error
	switch dispatch.Type {
	case DispatchTypeDefault:
		game = cluster.ChooseNode(cluster.RoleDefault)
	case DispatchTypeRole:
		game = cluster.ChooseNode(dispatch.Value)
		if game == nil {
			game = cluster.ChooseNode(cluster.RoleDefault)
		}
	case DispatchTypeInMap:
		meta, err := GetMeta(meta.ServerId)
		if err != nil {
			return nil, err
		}
		meta.GetNode()
	}
	return game, err
}

func MetaId(uuid string) string {
	return "Actor:" + uuid
}

func GenMetaId() string {
	return "Actor:" + xid.New().String()
}

func getFromEtcd(uuid string) (*Meta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	rsp, err := etcd.Client.Get(ctx, uuid)
	cancel()
	if err != nil {
		return nil, err
	}
	if rsp.Kvs == nil {
		return nil, nil
	}
	kv := rsp.Kvs[0]
	meta := &Meta{}
	if err := json.Unmarshal(kv.Value, meta); err != nil {
		return nil, err
	}
	meta.KV = kv
	meta.ModRevision = kv.ModRevision
	return setMetaCache(CacheMetas, meta), nil
}

func setToEtcd(meta *Meta) (*Meta, error) {
	data, err := json.Marshal(meta)
	if err != nil {
		return meta, err
	}
	// start transaction
	kv := clientv3.NewKV(etcd.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	txn := kv.Txn(ctx)
	cmp := clientv3.Compare(clientv3.ModRevision(meta.Uuid), "=", meta.ModRevision)
	putCmd := clientv3.OpPut(meta.Uuid, string(data))
	getCmd := clientv3.OpGet(meta.Uuid)
	txn.If(cmp).Then(putCmd).Else(getCmd)
	// commit
	txnRsp, err := txn.Commit()
	if err != nil {
		return meta, err
	}
	// process
	if txnRsp.Succeeded {
		meta.ModRevision = txnRsp.Header.Revision
		return setMetaCache(CacheMetas, meta), nil
	} else {
		kv := txnRsp.Responses[0].GetResponseRange().Kvs[0]
		return StoreMeta(CacheMetas, kv), nil
	}
}

func StoreMeta(metas map[string]*Meta, kv *mvccpb.KeyValue) *Meta {
	meta := &Meta{}
	err := json.Unmarshal(kv.Value, meta)
	if err == nil {
		meta.KV = kv
		meta.ModRevision = kv.ModRevision
		return setMetaCache(metas, meta)
	} else {
		logger.ERR("unmarshal meta failed: ", err, string(kv.Value))
	}
	return nil
}

var rwlock = &sync.RWMutex{}

func getMetaCache(uuid string) *Meta {
	rwlock.RLock()
	defer rwlock.RUnlock()
	if cache, ok := CacheMetas[uuid]; ok {
		return cache
	}
	return nil
}

func setMetaCache(metas map[string]*Meta, meta *Meta) *Meta {
	rwlock.Lock()
	defer rwlock.Unlock()
	if old, ok := metas[meta.Uuid]; ok {
		if old.ModRevision < meta.ModRevision {
			metas[meta.Uuid] = meta
		} else {
			meta = old
		}
	} else {
		metas[meta.Uuid] = meta
	}
	return meta
}

func DelMetaCache(uuid string, modRevision int64) {
	rwlock.Lock()
	defer rwlock.Unlock()
	if old, ok := CacheMetas[uuid]; ok {
		if old.ModRevision <= modRevision {
			delete(CacheMetas, uuid)
		}
	}
}

func daemonKey(nodeId string) string {
	return daemonPrefix() + nodeId
}

func daemonPrefix() string {
	return "DadmonMetas:"
}
