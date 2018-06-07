package router

import (
	"github.com/tiglabs/baudengine/proto/metapb"
	"github.com/tiglabs/baudengine/util/ttlcache"
	"time"
)

const (
	SPACE_EXPIRE_DURATION = 60 * time.Second
)

type DB struct {
	*metapb.DB
	spaceCache   *ttlcache.TTLCache  // key: space name, value: *Space
}

func NewDB(meta *metapb.DB) *DB {
	return &DB{
		DB:         meta,
		spaceCache: ttlcache.NewTTLCache(SPACE_EXPIRE_DURATION),
	}
}

func (db *DB) GetSpace(spaceName string) *Space {
	if space, ok := db.spaceCache.Get(spaceName); ok {
		return space.(*Space)
	}

	spaceMeta := GetMasterClientInstance(nil).GetSpace(db.ID, spaceName)
	if spaceMeta == nil {
		return nil
	}

	newSpace := NewSpace(spaceMeta)
	db.spaceCache.Put(spaceMeta.Name, newSpace)

	return newSpace
}
