package router

import "github.com/tiglabs/baudengine/proto/masterpb"

type PSClient struct {
    metaMgr *MetaManager
}

func NewPSClient(manager *MetaManager) *PSClient {
    return &PSClient{
        metaMgr: manager,
    }
}

type Routing struct {
    r       *masterpb.Route
    params  map[string]string
    rawBody string
}

func (pc *PSClient) Close() {

}

func (pc *PSClient) Delete(routing *Routing) {

}
func (pc *PSClient) Retrieve(routing *Routing) {

}
func (pc *PSClient) Update(routing *Routing) {

}

