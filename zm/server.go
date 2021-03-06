package zm

import (
	"context"
	"github.com/tiglabs/baudengine/topo"
	"github.com/tiglabs/baudengine/util/log"
	"sync"
	"time"
)

var (
	MineIsLeader = false
	LeaderNodeId = ""
)

type ZoneMaster struct {
	config *Config
	wg     sync.WaitGroup

	cluster   *Cluster
	apiServer *ApiServer
	rpcServer *RpcServer

	processorManager *ProcessorManager
	workerManager    *WorkerManager
	psRpcClient      PSRpcClient

	topoServer    *topo.TopoServer
	participation topo.MasterParticipation
}

func NewServer() *ZoneMaster {
	return new(ZoneMaster)
}

func (zm *ZoneMaster) Start(config *Config) error {
	zm.config = config

	topoServer, err := topo.OpenServer("etcd3", config.ClusterCfg.GlobalServerAddrs, config.ClusterCfg.GlobalRootDir)
	if err != nil {
		log.Error("topo.OpenServer() failed. err:[%v]", err)
		zm.Shutdown()
		return err
	}

	zm.topoServer = topoServer
	masterCtx, cancelMaster := context.WithCancel(context.Background())
	defer cancelMaster()
	zm.cluster = NewCluster(masterCtx, config, zm.topoServer)
	if err := zm.cluster.Start(); err != nil {
		log.Error("fail to start cluster. err:[%v]", err)
		zm.Shutdown()
		return err
	}

	zm.rpcServer = NewRpcServer(config, zm.cluster)
	if err := zm.rpcServer.Start(); err != nil {
		log.Error("fail to start rpc server. err:[%v]", err)
		zm.Shutdown()
		return err
	}

	zm.apiServer = NewApiServer(config, zm.cluster)
	if err := zm.apiServer.Start(); err != nil {
		log.Error("fail to start api server. err:[%v]", err)
		zm.Shutdown()
		return err
	}

	zm.psRpcClient = GetPSRpcClientSingle(config)

	zm.participation, err = zm.topoServer.NewMasterParticipation(config.ClusterCfg.ZoneID, config.ClusterCfg.CurNodeId)
	if err != nil {
		return err
	}

	for {
		_, err := zm.participation.WaitForMastership()
		switch err {
		case nil:
			MineIsLeader = true
			LeaderNodeId = config.ClusterCfg.CurNodeId
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				id, err := zm.participation.GetCurrentMasterID(ctx)
				cancel()
				if err != nil || id != config.ClusterCfg.CurNodeId {
					MineIsLeader = false
					LeaderNodeId = id
					break
				}
			}
			break
		case topo.ErrInterrupted:
			break
		default:
			//log.Errorf("Got error while waiting for master, will retry in 5s: %v", err)
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

func (zm *ZoneMaster) Shutdown() {
	if zm.apiServer != nil {
		zm.apiServer.Close()
		zm.apiServer = nil
	}
	if zm.rpcServer != nil {
		zm.rpcServer.Close()
		zm.rpcServer = nil
	}
	if zm.workerManager != nil {
		zm.workerManager.Shutdown()
		zm.workerManager = nil
	}
	if zm.processorManager != nil {
		zm.processorManager.Close()
		zm.processorManager = nil
	}
	if zm.psRpcClient != nil {
		zm.psRpcClient.Close()
		zm.psRpcClient = nil
	}
	if zm.cluster != nil {
		zm.cluster.Close()
		zm.cluster = nil
	}
	zm.participation.Stop()
}
