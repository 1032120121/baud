package router

import (
    "github.com/tiglabs/baudengine/util/ttlcache"
    "time"
)

const (
    DB_EXPIRE_DURATION = 300 * time.Second
)

type MetaManager struct {
    dbCache  *ttlcache.TTLCache // key: db name, value: *DB
}

func NewMetaManager() *MetaManager {
    return &MetaManager{
        dbCache:  ttlcache.NewTTLCache(DB_EXPIRE_DURATION),
    }
}

func (mm *MetaManager) GetDB(dbName string) *DB {
    if db, found := mm.dbCache.Get(dbName); found {
        return db.(*DB)
    }

    dbMeta := GetMasterClientInstance(nil).GetDB(dbName)
    if dbMeta == nil {
        return nil
    }

    newDB := NewDB(dbMeta)
    mm.dbCache.Put(dbName, newDB)

    return newDB
}
