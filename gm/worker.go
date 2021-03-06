package gm

import (
	"github.com/tiglabs/baudengine/proto/masterpb"
	"github.com/tiglabs/baudengine/proto/metapb"
	"github.com/tiglabs/baudengine/topo"
	"github.com/tiglabs/baudengine/util/log"
	"golang.org/x/net/context"
	"runtime/debug"
	"sync"
	"time"
)

type WorkerManager struct {
	ctx    context.Context
	cancel context.CancelFunc

	workers     map[string]Worker
	workersLock sync.RWMutex
	wg          sync.WaitGroup

	cluster *Cluster
}

func NewWorkerManager(cluster *Cluster) *WorkerManager {
	wm := &WorkerManager{
		workers: make(map[string]Worker),
		cluster: cluster,
	}
	wm.ctx, wm.cancel = context.WithCancel(context.Background())

	return wm
}

func (wm *WorkerManager) Start() error {
	wm.addWorker(NewSpaceStateTransitionWorker(wm.cluster))

	wm.workersLock.RLock()
	defer wm.workersLock.RUnlock()

	for _, worker := range wm.workers {
		wm.runWorker(worker)
	}

	log.Info("Worker manager has started")
	return nil
}

func (wm *WorkerManager) Close() {
	wm.cancel()
	wm.wg.Wait()

	wm.workersLock.RLock()
	defer wm.workersLock.RUnlock()

	wm.workers = nil

	log.Info("Worker manager has closed")
}

func (wm *WorkerManager) addWorker(worker Worker) {
	if worker == nil {
		return
	}

	wm.workersLock.Lock()
	defer wm.workersLock.Unlock()

	if _, ok := wm.workers[worker.getName()]; ok {
		log.Error("worker[%v] have already existed in worker manager.", worker.getName())
		return
	}

	wm.workers[worker.getName()] = worker
}

func (wm *WorkerManager) runWorker(worker Worker) {
	wm.wg.Add(1)
	go func() {
		defer wm.wg.Done()

		timer := time.NewTimer(worker.getInterval())
		defer timer.Stop()

		for {
			select {
			case <-wm.ctx.Done():
				return
			case <-timer.C:
				func() {
					log.Info("worker[%v] is running.", worker.getName())
					defer func() {
						if e := recover(); e != nil {
							log.Error("recover worker panic. e[%s] \nstack:[%s]", e, debug.Stack())
						}
					}()
					worker.run()
				}()
				timer.Reset(worker.getInterval())
			}
		}
	}()
}

type Worker interface {
	getName() string
	getInterval() time.Duration
	run()
}

type SpaceStateTransitionWorker struct {
	cluster *Cluster
}

func NewSpaceStateTransitionWorker(cluster *Cluster) *SpaceStateTransitionWorker {
	return &SpaceStateTransitionWorker{
		cluster: cluster,
	}
}

func (w *SpaceStateTransitionWorker) getName() string {
	return "Space-State-Transition-Worker"
}

func (w *SpaceStateTransitionWorker) getInterval() time.Duration {
	return time.Second * 60
}

func (w *SpaceStateTransitionWorker) run() {
	zonesMap, err := w.cluster.GetAllZonesMap()
	if err != nil {
		log.Error("GetAllZonesMap error, err:[%v]", err)
		return
	}
	zonesName, err := w.cluster.GetAllZonesName()
	if err != nil {
		log.Error("GetAllZonesName error, err:[%v]", err)
		return
	}
	partitionInfosCacheMap := make(map[metapb.PartitionID]*masterpb.PartitionInfo)

	// Get latest partitionInfo from different zones
	for _, zoneMap := range zonesMap {
		partitionIdsInZone, err := getPartitionIdsByZone(zoneMap.Name)
		if err != nil {
			log.Error("getPartitionIdsByZone error, err:[%v]", err)
			continue
		}
		for _, partitionIdInZone := range partitionIdsInZone {
			partitionInfo, err := getPartitionInfoByZone(zoneMap.Name, partitionIdInZone)
			if err != nil {
				log.Error("getPartitionInfoByZone error, err:[%v]", err)
				continue
			}
			if partitionInfo == nil {
				continue
			}
			partitionInfoInCacheMap := partitionInfosCacheMap[partitionInfo.ID]
			if partitionInfoInCacheMap == nil {
				partitionInfosCacheMap[partitionInfo.ID] = partitionInfo
				continue
			}
			if partitionInfo.Epoch.ConfVersion > partitionInfoInCacheMap.Epoch.ConfVersion {
				partitionInfosCacheMap[partitionInfo.ID] = partitionInfo
				continue
			} else if partitionInfo.Epoch.ConfVersion == partitionInfoInCacheMap.Epoch.ConfVersion {
				if partitionInfo.RaftStatus.Term > partitionInfoInCacheMap.RaftStatus.Term {
					partitionInfosCacheMap[partitionInfo.ID] = partitionInfo
					continue
				}
			}
		}
	}

	for partitionId, partitionInfoInCacheMap := range partitionInfosCacheMap {
		replicaLeaderMeta := pickLeaderReplica(partitionInfoInCacheMap)
		if replicaLeaderMeta == nil {
			continue
		}
		partition := w.cluster.PartitionCache.FindPartitionById(partitionId)
		if partition != nil {
			_, err := partition.updateReplicaGroup(partitionInfoInCacheMap)
			if err != nil {
				log.Error("updateReplicaGroup error, err:[%v]", err)
				continue
			}
		}
		// Compensation
		err := w.handleCompensation(partitionInfoInCacheMap, zonesName)
		if err != nil {
			log.Error("handleCompensation error, err:[%v]", err)
			continue
		}
	}

	dbs := w.cluster.DbCache.GetAllDBs()
	for _, db := range dbs {
		spaces := db.SpaceCache.GetAllSpaces()
		for _, space := range spaces {
			func() {
				space.propertyLock.Lock()
				defer space.propertyLock.Unlock()

				partitionsMap := space.partitions

				if space.Status == metapb.SS_Init {
					err := w.handleSpaceStateSSInit(db, space, partitionsMap, zonesName)
					if err != nil {
						log.Error("handleSpaceStateSSInit error, err:[%v]", err)
						return
					}
				} else if space.Status == metapb.SS_Running {
					err := w.handleSpaceStateSSRunning(db, space, partitionsMap, zonesName)
					if err != nil {
						log.Error("handleSpaceStateSSRunning error, err:[%v]", err)
						return
					}
				} else if space.Status == metapb.SS_Deleting {
					err := w.handleSpaceStateSSDeleting(db, space, partitionsMap, zonesName)
					if err != nil {
						log.Error("handleSpaceStateSSDeleting error, err:[%v]", err)
						return
					}
				}
			}()
		}
	}
}

func (w *SpaceStateTransitionWorker) handleCompensation(partitionInfo *masterpb.PartitionInfo, zonesName []string) error {
	partitionInCluster := w.cluster.PartitionCache.FindPartitionById(partitionInfo.ID)
	// partition元数据已经不存在, 但是partitionInfo存在
	if partitionInCluster == nil {
		log.Info("received a partition[%v], that not existed in cluster.", partitionInfo.ID)
		// force to delete
		replica := pickReplicaToDelete(partitionInfo)
		replicaZoneAddr, err := w.getZMLeaderAddr(replica.Zone, w.cluster.config.ClusterCfg.GmNodeId)
		if err != nil {
			log.Error("getZMLeaderAddr() replicaZoneAddr error. err:[%v]", err)
			return err
		}
		if replicaZoneAddr == "" {
			log.Info("getZMLeaderAddr() replicaZoneAddr has no leader now.")
			return ErrNoMSLeader
		}

		isGrabed, err := partitionInCluster.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partitionInCluster.ID))
		if err != nil {
			log.Error("partition grab Partition Task error, partition:[%d]", partitionInCluster.ID)
			return err
		}
		if !isGrabed {
			log.Info("partition has task now, partition:[%d]", partitionInCluster.ID)
			return nil
		}
		if err := GetPMSingle(w.cluster).PushEvent(NewPartitionForceDeleteEvent(replicaZoneAddr, partitionInCluster.ID)); err != nil {
			log.Error("fail to push event for force delete partition[%v].", partitionInCluster.ID)
			return err
		}
		return nil
	}

	log.Info("partition id[%v], confVerPartitionInfo[%v], confVerPartitionInCluster[%v]", partitionInCluster.ID, partitionInfo.Epoch.ConfVersion, partitionInCluster.Epoch.ConfVersion)
	// partitionInfo confver小于 partition元数据的confver, 说明这个版本的副本有问题, 应该删除
	if partitionInfo.Epoch.ConfVersion < partitionInCluster.Epoch.ConfVersion {
		// delete all replicas and leader
		replicaZoneAddr, replica, replicaLeaderZoneAddr, err := w.getReplicaZoneAddrAndRelicaLeaderZoneAddrForDelete(nil, partitionInfo)
		if err != nil {
			log.Error("getReplicaZoneAddrAndRelicaLeaderZoneAddr error, err:[%v]", err)
			return err
		}
		isGrabed, err := partitionInCluster.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partitionInCluster.ID))
		if err != nil {
			log.Error("partition grab Partition Task error, partition:[%d]", partitionInCluster.ID)
			return err
		}
		if !isGrabed {
			log.Info("partition has task now, partition:[%d]", partitionInCluster.ID)
			return nil
		}
		if err := GetPMSingle(w.cluster).PushEvent(NewPartitionDeleteEvent(replicaZoneAddr, replicaLeaderZoneAddr, partitionInfo.ID, replica)); err != nil {
			log.Error("fail to push event for deleting partitionInfo[%v].", partitionInfo)
			return err
		}
		return nil
	} else if partitionInfo.Epoch.ConfVersion == partitionInCluster.Epoch.ConfVersion {
		if partitionInCluster.ReplicaLeader != nil && partitionInfo.RaftStatus.Replica.ID == partitionInCluster.ReplicaLeader.ID {
			return nil
		}
		if partitionInCluster.ReplicaLeader != nil && partitionInfo.RaftStatus.Replica.ID != partitionInCluster.ReplicaLeader.ID {
			if partitionInfo.RaftStatus.Term < partitionInCluster.Term {
				// delete all replicas and leader
				replicaZoneAddr, replica, replicaLeaderZoneAddr, err := w.getReplicaZoneAddrAndRelicaLeaderZoneAddrForDelete(nil, partitionInfo)
				if err != nil {
					log.Error("getReplicaZoneAddrAndRelicaLeaderZoneAddr error, err:[%v]", err)
					return err
				}
				isGrabed, err := partitionInCluster.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partitionInCluster.ID))
				if err != nil {
					log.Error("partition grab Partition Task error, partition:[%d]", partitionInCluster.ID)
					return err
				}
				if !isGrabed {
					log.Info("partition has task now, partition:[%d]", partitionInCluster.ID)
					return nil
				}
				if err := GetPMSingle(w.cluster).PushEvent(NewPartitionDeleteEvent(replicaZoneAddr, replicaLeaderZoneAddr, partitionInfo.ID, replica)); err != nil {
					log.Error("fail to push event for deleting partitionInfo[%v].", partitionInfo)
					return err
				}
				return nil
			}
		}
	}
	return nil
}

func (w *SpaceStateTransitionWorker) handleSpaceStateSSInit(db *DB, space *Space, partitionsMap map[metapb.PartitionID]*Partition, zonesName []string) error {
	var isSpaceReady = true
	if partitionsMap == nil || len(partitionsMap) == 0 {
		log.Error("space has no partition, db:[%s], space:[%s]", db.Name, space.Name)
		return nil
	}
	for _, partition := range partitionsMap {
		if partition.countReplicas() < FIXED_REPLICA_NUM {
			isSpaceReady = false
			replicaZoneAddr, replicaZoneName, replicaLeaderZoneAddr, err := w.getReplicaZoneAddrAndRelicaLeaderZoneAddrForCreate(zonesName, partition)
			if err != nil {
				log.Error("getReplicaZoneAddrAndRelicaLeaderZoneAddr error, err:[%v]", err)
				continue
			}
			isGrabed, err := partition.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partition.ID))
			if err != nil {
				log.Error("partition grab Partition Task error, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
				continue
			}
			if !isGrabed {
				log.Info("partition has task now, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
				continue
			}
			if err := GetPMSingle(w.cluster).PushEvent(NewPartitionCreateEvent(replicaZoneAddr, replicaZoneName, replicaLeaderZoneAddr, partition)); err != nil {
				log.Error("fail to push event for creating partition[%v].", partition)
				continue
			}
		}
	}
	if isSpaceReady {
		space.Status = metapb.SS_Running
		space.update()
	}
	return nil
}

func (w *SpaceStateTransitionWorker) handleSpaceStateSSRunning(db *DB, space *Space, partitionsMap map[metapb.PartitionID]*Partition, zonesName []string) error {
	if partitionsMap == nil || len(partitionsMap) == 0 {
		log.Error("space has no partition, db:[%s], space:[%s]", db.Name, space.Name)
		return nil
	}
	for _, partition := range partitionsMap {
		if partition.countReplicas() != FIXED_REPLICA_NUM {
			if partition.countReplicas() < FIXED_REPLICA_NUM {
				replicaZoneAddr, replicaZoneName, replicaLeaderZoneAddr, err := w.getReplicaZoneAddrAndRelicaLeaderZoneAddrForCreate(zonesName, partition)
				if err != nil {
					log.Error("getReplicaZoneAddrAndRelicaLeaderZoneAddr error, err:[%v]", err)
					continue
				}
				isGrabed, err := partition.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partition.ID))
				if err != nil {
					log.Error("Partition grab Partition Task error, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
					continue
				}
				if !isGrabed {
					log.Info("partition has task now, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
					continue
				}
				if err := GetPMSingle(w.cluster).PushEvent(NewPartitionCreateEvent(replicaZoneAddr, replicaZoneName, replicaLeaderZoneAddr, partition)); err != nil {
					log.Error("fail to push event for creating partition[%v].", partition)
					continue
				}
			} else if partition.countReplicas() > FIXED_REPLICA_NUM {
				replicaZoneAddr, replica, replicaLeaderZoneAddr, err := w.getReplicaZoneAddrAndRelicaLeaderZoneAddrForDelete(partition, nil)
				if err != nil {
					log.Error("getReplicaZoneAddrAndRelicaLeaderZoneAddr error, err:[%v]", err)
					continue
				}
				isGrabed, err := partition.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partition.ID))
				if err != nil {
					log.Error("partition grab Partition Task error, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
					continue
				}
				if !isGrabed {
					log.Info("partition has task now, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
					continue
				}
				if err := GetPMSingle(w.cluster).PushEvent(NewPartitionDeleteEvent(replicaZoneAddr, replicaLeaderZoneAddr, partition.ID, replica)); err != nil {
					log.Error("fail to push event for deleting partition[%v].", partition)
					continue
				}
			}
		}
	}
	return nil
}

func (w *SpaceStateTransitionWorker) handleSpaceStateSSDeleting(db *DB, space *Space, partitionsMap map[metapb.PartitionID]*Partition, zonesName []string) error {
	var isSpaceCanDelete = true
	if partitionsMap == nil || len(partitionsMap) == 0 {
		log.Error("space has no partition, db:[%s], space:[%s]", db.Name, space.Name)
		return nil
	}
	for _, partition := range partitionsMap {
		if partition.countReplicas() > 0 {
			isSpaceCanDelete = false
			replicaZoneAddr, replica, replicaLeaderZoneAddr, err := w.getReplicaZoneAddrAndRelicaLeaderZoneAddrForDelete(partition, nil)
			if err != nil {
				log.Error("getReplicaZoneAddrAndRelicaLeaderZoneAddr error, err:[%v]", err)
				continue
			}
			isGrabed, err := partition.grabPartitionTaskLock(topo.GlobalZone, "partition", string(partition.ID))
			if err != nil {
				log.Error("partition grab Partition Task error, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
				continue
			}
			if !isGrabed {
				log.Info("partition has task now, db:[%s], space:[%s], partition:[%d]", db.Name, space.Name, partition.ID)
				continue
			}
			if err := GetPMSingle(w.cluster).PushEvent(NewPartitionDeleteEvent(replicaZoneAddr, replicaLeaderZoneAddr, partition.ID, replica)); err != nil {
				log.Error("fail to push event for deleting partition[%v].", partition)
				continue
			}
		}
	}
	if isSpaceCanDelete {
		space.Status = metapb.SS_Delete
		space.erase()
	}
	return nil
}

func (w *SpaceStateTransitionWorker) getReplicaZoneAddrAndRelicaLeaderZoneAddrForCreate(zonesName []string, partition *Partition) (string, string, string, error) {
	replicaZoneName := NewZoneSelector().SelectTarget(zonesName)
	var replicaLeaderZone string
	if partition.ReplicaLeader != nil {
		replicaLeaderZone = partition.ReplicaLeader.Zone
	} else {
		replicaLeaderZone = replicaZoneName
	}

	replicaZoneAddr, err := w.getZMLeaderAddr(replicaZoneName, w.cluster.config.ClusterCfg.GmNodeId)
	if err != nil {
		log.Error("getZMLeaderAddr() replicaZoneAddr error. err:[%v]", err)
		return "", "", "", err
	}
	if replicaZoneAddr == "" {
		log.Info("getZMLeaderAddr() replicaZoneAddr has no leader now.")
		return "", "", "", ErrNoMSLeader
	}

	replicaLeaderZoneAddr, err := w.getZMLeaderAddr(replicaLeaderZone, w.cluster.config.ClusterCfg.GmNodeId)
	if err != nil {
		log.Error("getZMLeaderAddr() replicaLeaderZoneAddr error. err:[%v]", err)
		return "", "", "", err
	}
	if replicaLeaderZoneAddr == "" {
		log.Info("getZMLeaderAddr() replicaLeaderZoneAddr has no leader now.")
		return "", "", "", ErrNoMSLeader
	}
	return replicaZoneAddr, replicaZoneName, replicaLeaderZoneAddr, nil
}

func (w *SpaceStateTransitionWorker) getReplicaZoneAddrAndRelicaLeaderZoneAddrForDelete(partition *Partition, partitionInfo *masterpb.PartitionInfo) (string, *metapb.Replica, string, error) {
	var replicaLeaderZone string
	var replica *metapb.Replica
	if partition != nil {
		log.Debug("getAddr by partition")
		if partition.ReplicaLeader == nil {
			log.Info("partition has no leader now.")
			return "", nil, "", ErrNoMSLeader
		}
		replicaLeaderZone = partition.ReplicaLeader.Zone
		replica = partition.pickReplicaToDelete()
	}
	if partitionInfo != nil {
		log.Debug("getAddr by partitionInfo")
		if !partitionInfo.IsLeader {
			log.Info("partitionInfo has no leader now.")
			return "", nil, "", ErrNoMSLeader
		}
		replicaLeaderZone = partitionInfo.RaftStatus.Replica.Zone
		replica = pickReplicaToDelete(partitionInfo)
	}
	replicaZoneName := replica.Zone
	replicaZoneAddr, err := w.getZMLeaderAddr(replicaZoneName, w.cluster.config.ClusterCfg.GmNodeId)
	if err != nil {
		log.Error("getZMLeaderAddr() replicaZoneAddr error. err:[%v]", err)
		return "", nil, "", err
	}
	if replicaZoneAddr == "" {
		log.Info("getZMLeaderAddr() replicaZoneAddr has no leader now.")
		return "", nil, "", ErrNoMSLeader
	}

	replicaLeaderZoneAddr, err := w.getZMLeaderAddr(replicaLeaderZone, w.cluster.config.ClusterCfg.GmNodeId)
	if err != nil {
		log.Error("getZMLeaderAddr() replicaLeaderZoneAddr error. err:[%v]", err)
		return "", nil, "", err
	}
	if replicaLeaderZoneAddr == "" {
		log.Info("getZMLeaderAddr() replicaLeaderZoneAddr has no leader now.")
		return "", nil, "", ErrNoMSLeader
	}
	return replicaZoneAddr, replica, replicaLeaderZoneAddr, nil
}

func (w *SpaceStateTransitionWorker) getZMLeaderAddr(zoneName, id string) (string, error) {
	replicaZoneParticipation, err := TopoServer.NewMasterParticipation(zoneName, id)
	if err != nil {
		log.Error("TopoServer NewMasterParticipation error. err:[%v]", err)
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), ETCD_TIMEOUT)
	replicaZoneAddr, err := replicaZoneParticipation.GetCurrentMasterID(ctx)
	cancel()
	if err != nil {
		log.Error("replicaZoneParticipation GetCurrentMasterID error. err:[%v]", err)
		return "", err
	}
	return replicaZoneAddr, nil
}
