package router

import (
	"github.com/tiglabs/baudengine/proto/masterpb"
	"github.com/tiglabs/baudengine/proto/metapb"
	"sort"
	"sync"
)

type Space struct {
	*metapb.Space
	partitions []*Partition
	lock       sync.RWMutex
}

func NewSpace(spaceMeta *metapb.Space) *Space {
	return &Space{
		Space:      spaceMeta,
		partitions: make([]*Partition, 0),
	}
}

func (space *Space) GetPartition(slotId metapb.SlotID) *Partition {
	partition, _ := space.getPartition(slotId)
	if partition == nil {
		routes := GetMasterClientInstance(nil).GetRoute(space.DB, space.ID, slotId)
		space.addRoutes(routes)
		partition, _ = space.getPartition(slotId)
	}

	return partition
}

func (space *Space) GetKeyField() string {
	return space.KeyPolicy.KeyField
}

func (space *Space) Delete(partition *metapb.Partition) {
	_, pos := space.getPartition(partition.StartSlot)
	if pos >= 0 {
		space.lock.Lock()
		defer space.lock.Unlock()

		space.partitions = append(space.partitions[:pos], space.partitions[pos + 1:]...)
	}
}

func (space *Space) getPartition(slotId metapb.SlotID) (*Partition, int) {
	space.lock.RLock()
	defer space.lock.RUnlock()

	pos := sort.Search(len(space.partitions), func(i int) bool {
		return space.partitions[i].EndSlot >= slotId
	})
	if pos >= len(space.partitions) || slotId < space.partitions[pos].StartSlot {
		return nil, -1
	}
	return space.partitions[pos], pos
}

func (space *Space) addRoutes(routes []masterpb.Route) {
	space.lock.Lock()
	defer space.lock.Unlock()

	for _, route := range routes {
		pos := sort.Search(len(space.partitions), func(i int) bool {
			return space.partitions[i].EndSlot >= route.StartSlot
		})
		newPartition := NewPartition(&route.Partition, &route)
		if pos >= len(space.partitions) {
			space.partitions = append(space.partitions, newPartition)
		} else if space.partitions[pos].EndSlot >= route.StartSlot {
			space.partitions = append(space.partitions[:pos], append([]*Partition{newPartition}, space.partitions[pos:]...)...)
		}
	}
}
