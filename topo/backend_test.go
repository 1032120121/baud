package topo
//
//import (
//    "context"
//    "time"
//    "sync"
//    "encoding/gob"
//)
//
//type backendMock struct {
//    zoneCache map[string]*nodeCacheMock
//    lock      sync.RWMutex
//}
//
//func NewBackendMock(zones []string) *backendMock {
//    zoneCache := make(map[string]*nodeCacheMock)
//    for _, zone := range zones {
//       zoneCache := &nodeCacheMock{
//           contents: make(map[string][]byte),
//           versions: make(map[string]int64),
//       }
//    }
//    return &backendMock{
//        zoneCache: make(map[string]*nodeCacheMock),
//    }
//}
//
//type nodeCacheMock struct {
//    contents map[string][]byte
//    versions map[string]int64
//    lock     sync.RWMutex
//}
//
//func (m *backendMock) Close() {
//}
//
//func (m *backendMock) ListDir(ctx context.Context, cell, dirPath string) ([]string, Version, error) {
//
//}
//
//func (m *backendMock) WatchDir(ctx context.Context, cell, dirPath string, version Version) (<-chan *WatchData, CancelFunc, error) {
//
//}
//
//func (m *backendMock) Create(ctx context.Context, cell, filePath string, contents []byte) (Version, error) {
//    m.lock.Lock()
//    defer m.lock.Unlock()
//
//    cache, ok := m.zoneCache[cell]
//}
//
//func (m *backendMock) CreateUniqueEphemeral(ctx context.Context, cell string, filePath string, contents []byte, timeout time.Duration) (Version, error) {
//
//}
//
//func (m *backendMock) Update(ctx context.Context, cell, filePath string, contents []byte, version Version) (Version, error) {
//
//}
//
//func (m *backendMock) Get(ctx context.Context, cell, filePath string) ([]byte, Version, error) {
//
//}
//
//func (m *backendMock) Delete(ctx context.Context, cell, filePath string, version Version) error {
//
//}
//
//func (m *backendMock) Watch(ctx context.Context, cell, filePath string) (current *WatchData, changes <-chan *WatchData, cancel CancelFunc) {
//
//}
//
//func (m *backendMock) NewMasterParticipation(cell, id string) (MasterParticipation, error) {
//
//}
//
//func (m *backendMock) NewTransaction(ctx context.Context, cell string) (Transaction, error) {
//
//}