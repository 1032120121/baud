package topo
//
//import (
//    "testing"
//    "context"
//    "github.com/tiglabs/baudengine/proto/metapb"
//    "github.com/tiglabs/baudengine/util/assert"
//)
//
//const (
//    ZONE1_NAME = "zone1"
//    ZONE1_ADDR = "zone1_addrs"
//    ZONE1_ROOT = "/zone1_root"
//)
//
//func TestAddZone(t *testing.T) {
//    ctx := context.Context(context.Background())
//    server := newTopoServerMock()
//
//    zone := &metapb.Zone{Name: ZONE1_NAME, ServerAddrs: ZONE1_ADDR, RootDir: ZONE1_ROOT}
//    zoneTopo, err := server.AddZone(ctx, zone)
//
//    assert.Nil(t, err)
//    assert.NotNil(t, zoneTopo)
//
//    assert.Equal(t, zoneTopo.Name, ZONE1_NAME, "unmatched zone name")
//    assert.Equal(t, zoneTopo.ServerAddrs, ZONE1_ADDR, "unmatched zone server addr")
//    assert.Equal(t, zoneTopo.RootDir, ZONE1_ROOT, "unmatched zone root dir")
//    assert.Greater(t, zoneTopo.Version, 0)
//}
//
//func newTopoServerMock() *TopoServer {
//    zones := make([]string, 0)
//    zones = append(zones, GlobalZone)
//    zones = append(zones, ZONE1_NAME)
//
//    return &TopoServer{
//        backend:  NewBackendMock(zones),
//    }
//}
