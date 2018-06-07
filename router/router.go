package router

import (
	"github.com/tiglabs/baudengine/util/log"
)

type Router struct {
	config       *Config
	apiServer    *ApiServer
	masterClient *MasterClient
	metaMgr      *MetaManager
}

func NewServer() *Router {
	return new(Router)
}

func (router *Router) Start(cfg *Config) error {
	router.config = cfg

	router.masterClient = GetMasterClientInstance(cfg)
	if router.masterClient == nil {
		log.Error("Fail to create master client.")
		return ErrInternalError
	}

	router.metaMgr = NewMetaManager()

	router.apiServer = NewApiServer(cfg, router.metaMgr)
	if err := router.apiServer.Start(); err != nil {
		log.Error("Fail to start api server")
		return err
	}

	return nil
}

func (router *Router) Shutdown() {
	if router.apiServer != nil {
		router.apiServer.Shutdown()
		router.apiServer = nil
	}
	if router.masterClient != nil {
		router.masterClient.Close()
		router.masterClient = nil
	}
}


//
//func (router *Router) handleRead(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {
//	defer router.catchPanic(writer)
//
//	_, _, partition, docId := router.getParams(params, true)
//	docBody := partition.Read(docId)
//	sendReply(writer, &HttpReply{ERRCODE_SUCCESS, ErrSuccess.Error(), docBody})
//}
//
//func (router *Router) handleUpdate(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {
//	defer router.catchPanic(writer)
//
//	_, _, partition, docId := router.getParams(params, true)
//	docBody := router.readDocBody(request)
//	partition.Update(docId, docBody)
//	sendReply(writer, &HttpReply{ERRCODE_SUCCESS, ErrSuccess.Error(), nil})
//}
//
//func (router *Router) handleDelete(writer http.ResponseWriter, request *http.Request, params netutil.UriParams) {
//	defer router.catchPanic(writer)
//
//	_, _, partition, docId := router.getParams(params, true)
//	if ok := partition.Delete(docId); ok {
//		sendReply(writer, &HttpReply{ERRCODE_SUCCESS, ErrSuccess.Error(), nil})
//	} else {
//		sendReply(writer, &HttpReply{ERRCODE_INTERNAL_ERROR, "Cannot delete doc", nil})
//	}
//}
//
//func (router *Router) getParams(params netutil.UriParams, decodeDocId bool) (db *DB, space *Space, partition *Partition, docId *metapb.DocID) {
//	defer func() {
//		if p := recover(); p != nil {
//			if err, ok := p.(error); ok {
//				log.Error("getParams() failed: %s", err.Error())
//			}
//			panic(&HttpReply{ERRCODE_PARAM_ERROR, ErrParamError.Error(), nil})
//		}
//	}()
//
//	db = router.GetDB(params.ByName("db"))
//	space = db.GetSpace(params.ByName("space"))
//	if decodeDocId {
//		id, err := keys.DecodeDocIDFromString(params.ByName("docId"))
//		if err != nil {
//			panic(err)
//		}
//		docId = id
//	}
//	return
//}

//func (router *Router) readDocBody(request *http.Request) []byte {
//	var docBody = make([]byte, request.ContentLength)
//	request.Body.Read(docBody)
//	return docBody
//}

//func (router *Router) catchPanic(writer http.ResponseWriter) {
//	if p := recover(); p != nil {
//		switch t := p.(type) {
//		case *HttpReply:
//			sendReply(writer, t)
//		case error:
//			sendReply(writer, &HttpReply{ERRCODE_INTERNAL_ERROR, t.Error(), nil})
//			log.Error("catchPanic() error: %s", t.Error())
//		default:
//			sendReply(writer, &HttpReply{ERRCODE_INTERNAL_ERROR, ErrInternalError.Error(), nil})
//		}
//	}
//}
