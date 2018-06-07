package router

import (
    "github.com/tiglabs/baudengine/util/netutil"
    "strconv"
    "github.com/tiglabs/baudengine/proto/metapb"
    "net/http"
    "github.com/tiglabs/baudengine/util/log"
    "github.com/tiglabs/baudengine/util/json"
    "github.com/spaolacci/murmur3"
    "fmt"
    "github.com/tiglabs/baudengine/util"
    "time"
    "io/ioutil"
)

const (
    DEFAULT_CONN_MAX_LIMIT = 10000
    DEFAULT_CLOSE_TIMEOUT  = 5 * time.Second
)

const (
    URI_DB      = "db"
    URI_SPACE   = "space"
    URI_DOCID   = "docid"
    URI_SPACES  = "spaces"
    URI_DBS     = "dbs"
    URI_ALL     = "_all"

    URL_ROUTING = "routing"
)

type ApiServer struct {
    config   *Config
    server   *netutil.Server
    metaMgr  *MetaManager
    psClient *PSClient
}

func NewApiServer(cfg *Config, manager *MetaManager) *ApiServer {
    apiServer := &ApiServer{}
    apiServer.config = cfg
    apiServer.metaMgr = manager
    apiServer.psClient = NewPSClient(manager)

    httpServerConfig := &netutil.ServerConfig{
        Name:         "gm-api-server",
        Addr:         util.BuildAddr("0.0.0.0", cfg.ModuleCfg.HttpPort),
        Version:      "v1",
        ConnLimit:    DEFAULT_CONN_MAX_LIMIT,
        CloseTimeout: DEFAULT_CLOSE_TIMEOUT,
    }

    httpServer := netutil.NewServer(httpServerConfig)
    // /:db/:space/:docId/_create
    httpServer.Handle(netutil.PUT, fmt.Sprintf("/:%s/:%s/:%s/_create", URI_DB, URI_SPACE, URI_DOCID), apiServer.handleCreate)
    // /:db/:space
    httpServer.Handle(netutil.POST, fmt.Sprintf("/:%s/:%s", URI_DB, URI_SPACE), apiServer.handleCreateAutoId)
    httpServer.Handle(netutil.PUT, "/:db/:space/:docId", apiServer.handleUpsert)
    httpServer.Handle(netutil.POST, "/:db/:space/:docId/_update", apiServer.handleUpdate)
    httpServer.Handle(netutil.DELETE, "/:db/:space/:docId", apiServer.handleDelete)
    httpServer.Handle(netutil.GET, "/:db/:space/:docId", apiServer.handleRead)
    httpServer.Handle(netutil.GET, "/:db/_search", apiServer.handleSearchByDB)
    httpServer.Handle(netutil.GET, "/:db/:spaces/_search", apiServer.handleSearchBySpaces)
    httpServer.Handle(netutil.GET, "/:dbs/:space/_search", apiServer.handleSearchByDBs)
    httpServer.Handle(netutil.GET, "/_search", apiServer.handleSearch)
    apiServer.server = httpServer

    return apiServer
}

func (s *ApiServer) Start() error {
     return s.server.Run()
}

func (s *ApiServer) Shutdown() {
    if s.psClient != nil {
        s.psClient.Close()
        s.psClient = nil
    }
    if s.server != nil {
        s.server.Close()
        s.server = nil
    }
}

func (s *ApiServer) handleCreate(w http.ResponseWriter, req *http.Request, params netutil.UriParams) {
    // defer router.catchPanic(writer)

    dbName := params.ByName(URI_DB)
    db := s.metaMgr.GetDB(dbName)
    if db == nil {
        sendReply(w, newHttpErrReply(ErrDbNotExists))
        return
    }

    spaceName := params.ByName(URI_SPACE)
    space := db.GetSpace(spaceName)
    if space == nil {
        sendReply(w, newHttpErrReply(ErrSpaceNotExists))
        return
    }

    docId := params.ByName(URI_DOCID)
    if len(docId) == 0 {
        sendReply(w, newHttpErrReply(ErrUriError))
        return
    }

    routingVal := req.FormValue(URL_ROUTING)
    if len(routingVal) == 0 {
        routingVal = parseRoutingVal(req, space.GetKeyField())
        if len(routingVal) == 0 {
            log.Error("Not found routing field vale")
            sendReply(w, newHttpErrReply(ErrParamError))
            return
        }
    }
    slotId := calcSlotID(routingVal)

    partition := space.GetPartition(slotId)
    if partition == nil {
        sendReply(w, newHttpErrReply(ErrPartitionNotExists))
        return
    }

    routing := &Routing{
        r: partition.route,
        params:parseReqParams(req),
        rawBody: parseReqBody(req),
    }
    s.psClient.Update(routing)

}

func (s *ApiServer) handleCreateAutoId(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {
}

func (s *ApiServer) handleUpsert(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {
}
func (s *ApiServer) handleUpdate(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {
}

func (s *ApiServer) handleDelete(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {}

func (s *ApiServer) handleRead(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {}

func (s *ApiServer) handleSearchByDB(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {}

func (s *ApiServer) handleSearchBySpaces(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {}

func (s *ApiServer) handleSearchByDBs(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {}

func (s *ApiServer) handleSearch(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {}

func parseRoutingVal(req *http.Request, keyField string) string {
    if len(keyField) == 0 {
        return ""
    }

    body := parseReqBody(req)
    if len(body) == 0 {
        return ""
    }

    m := make(map[string]interface{})
    if err := json.Unmarshal([]byte(body), &m); err != nil {
        log.Error("unmarshal request body error[%v]", err)
        return ""
    }

    value, ok := m[keyField]
    if !ok {
        return ""
    }

    return value.(string)
}

func parseReqBody(req *http.Request) string {
    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        log.Error("Read request body error[%v]", err)
        return ""
    }
    if body == nil {
        return ""
    }
    return string(body)
}

func parseReqParams(req *http.Request) map[string]string {
    params := make(map[string]string)
    for name, values := range req.Form {
        if len(values) > 0 {
            params[name] = values[0]
        } else {
            params[name] = ""
        }
    }
    return params
}

func calcSlotID(data string) metapb.SlotID {
    return metapb.SpaceID(murmur3.Sum32WithSeed([]byte(data), 0))
}

type HttpReply struct {
    Code int32       `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data,omitempty"`
}

func newHttpSucReply(data interface{}) *HttpReply {
    return &HttpReply{
        Code: ERRCODE_SUCCESS,
        Msg:  ErrSuc.Error(),
        Data: data,
    }
}

func newHttpErrReply(err error) *HttpReply {
    if err == nil {
        return newHttpSucReply("")
    }

    code, ok := Err2CodeMap[err]
    if ok {
        return &HttpReply{
            Code: code,
            Msg:  err.Error(),
        }
    } else {
        return &HttpReply{
            Code: ERRCODE_INTERNAL_ERROR,
            Msg:  ErrInternalError.Error(),
        }
    }
}

func sendReply(w http.ResponseWriter, httpReply *HttpReply) {
    reply, err := json.Marshal(httpReply)
    if err != nil {
        log.Error("fail to marshal http reply[%v]. err:[%v]", httpReply, err)
        sendReply(w, newHttpErrReply(ErrInternalError))
        return
    }
    w.Header().Set("content-type", "application/json")
    w.Header().Set("Content-Length", strconv.Itoa(len(reply)))
    if _, err := w.Write(reply); err != nil {
        log.Error("fail to write http reply[%s] len[%d]. err:[%v]", string(reply), len(reply), err)
    }
}
