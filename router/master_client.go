package router

import (
	"context"
	"github.com/pkg/errors"
	"github.com/tiglabs/baudengine/proto/masterpb"
	"github.com/tiglabs/baudengine/proto/metapb"
	"github.com/tiglabs/baudengine/util/rpc"
	"google.golang.org/grpc"
	"github.com/tiglabs/baudengine/util/log"
	"github.com/tiglabs/baudengine/util/json"
	"sync"
)

var (
	mcOnce   sync.Once
	mcSingle *MasterClient
)

type MasterClient struct {
	config     *Config
	client     *rpc.Client
	masterAddr string
	context    context.Context
	cancelFunc context.CancelFunc
}

func GetMasterClientInstance(cfg *Config) *MasterClient {
	if mcSingle != nil {
		return mcSingle
	}

	mcOnce.Do(func() {
		if cfg == nil {
			log.Error("Config should not be empty when create master client!")
			return
		}
		mcSingle = newMasterClient(cfg)
	})
	return mcSingle
}

func newMasterClient(cfg *Config) *MasterClient {
	mc := &MasterClient{masterAddr: cfg.ModuleCfg.MasterAddr}
	connMgrOpt := rpc.DefaultManagerOption
	mc.context, mc.cancelFunc = context.WithCancel(context.Background())
	connMgr := rpc.NewConnectionMgr(mc.context, &connMgrOpt)
	clientOpt := rpc.DefaultClientOption
	clientOpt.ClusterID = cfg.ModuleCfg.ClusterId
	clientOpt.ConnectMgr = connMgr
	clientOpt.CreateFunc = func(clientConn *grpc.ClientConn) interface{} {
		return masterpb.NewMasterRpcClient(clientConn)
	}
	mc.client = rpc.NewClient(1, &clientOpt)
	return mc
}

func (mc *MasterClient) Close() {
	if mc.client != nil {
		mc.client.Close()
		mc.client = nil
	}
	mcSingle = nil
}

func (mc *MasterClient) GetRoute(dbId metapb.DBID, spaceId metapb.SpaceID, slotId metapb.SlotID) []masterpb.Route {
	request := &masterpb.GetRouteRequest{DB: dbId, Space: spaceId, Slot: slotId}
	ctx, cancel := mc.getContext()
	defer cancel()
	resp, err := mc.getClient().GetRoute(ctx, request)
	mc.checkResponseOk(&resp.ResponseHeader, err)
	text, err := json.Marshal(resp)
	if err == nil {
		log.Debug("GetRoute(slotId=%d) %s", slotId, string(text))
	}
	return resp.Routes
}

func (mc *MasterClient) GetDB(dbName string) *metapb.DB {
	request := &masterpb.GetDBRequest{DBName: dbName}
	ctx, cancel := mc.getContext()
	defer cancel()
	resp, err := mc.getClient().GetDB(ctx, request)
	mc.checkResponseOk(&resp.ResponseHeader, err)
	return &resp.Db
}

func (mc *MasterClient) GetSpace(id metapb.DBID, spaceName string) *metapb.Space {
	request := &masterpb.GetSpaceRequest{ID: id, SpaceName: spaceName}
	ctx, cancel := mc.getContext()
	defer cancel()
	resp, err := mc.getClient().GetSpace(ctx, request)
	mc.checkResponseOk(&resp.ResponseHeader, err)
	return &resp.Space
}

func (mc *MasterClient) getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(mc.context, rpcTimeoutDef)
}

func (mc *MasterClient) getClient() masterpb.MasterRpcClient {
	client, err := mc.client.GetGrpcClient(mc.masterAddr)
	if err != nil {
		log.Error("get master client for %s failed", mc.masterAddr)
		panic(err)
	}
	return client.(masterpb.MasterRpcClient)
}

func (mc *MasterClient) checkResponseOk(header *metapb.ResponseHeader, err error) {
	if err != nil {
		panic(err)
	}
	if header.Code != metapb.RESP_CODE_OK {
		if header.Code == metapb.MASTER_RESP_CODE_NOT_LEADER {
			mc.masterAddr = header.Error.NotLeader.LeaderAddr
		}
		panic(errors.New(header.Message))
	}
}
