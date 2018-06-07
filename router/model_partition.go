package router

import (
	"github.com/tiglabs/baudengine/proto/masterpb"
	"github.com/tiglabs/baudengine/proto/metapb"
)

type Partition struct {
	*metapb.Partition
	route *masterpb.Route
}

func NewPartition(partition *metapb.Partition, route *masterpb.Route) *Partition {
	return &Partition{
		Partition: partition,
		route:     route,
	}
}

//func (partition *Partition) Create(docBody []byte) *metapb.DocID {
//	createReq := pspb.BulkItemRequest{
//		OpType: pspb.OpType_CREATE,
//		Index:  &pspb.IndexRequest{Source: docBody},
//	}
//	request := &pspb.BulkRequest{
//		ActionRequestHeader: partition.requestHeader,
//		Requests:            []pspb.BulkItemRequest{createReq},
//	}
//	ctx, cancel := partition.getContext()
//	defer cancel()
//	resp := partition.getSingleResponse(partition.getClient().BulkWrite(ctx, request)).Index
//	docId, err := keys.DecodeDocIDFromString(resp.Id)
//	if err != nil {
//		panic(err)
//	}
//	return docId
//}
//
//func (partition *Partition) Read(docId *metapb.DocID) metapb.Value {
//	request := &pspb.GetRequest{ActionRequestHeader: partition.requestHeader, Id:*docId}
//	ctx, cancel := partition.getContext()
//	defer cancel()
//	resp, err := partition.getClient().Get(ctx, request)
//	if err != nil {
//		panic(err)
//	}
//	return resp.Fields
//}
//
//func (partition *Partition) Update(docId *metapb.DocID, docBody []byte) {
//	createReq := pspb.BulkItemRequest{
//		OpType: pspb.OpType_UPDATE,
//		Update: &pspb.UpdateRequest{Id: *docId, Doc: docBody},
//	}
//	request := &pspb.BulkRequest{
//		ActionRequestHeader: partition.requestHeader,
//		Requests:            []pspb.BulkItemRequest{createReq},
//	}
//	ctx, cancel := partition.getContext()
//	defer cancel()
//	resp := partition.getSingleResponse(partition.getClient().BulkWrite(ctx, request)).Index
//	docId, err := keys.DecodeDocIDFromString(resp.Id)
//	if err != nil {
//		panic(err)
//	}
//}
//
//func (partition *Partition) Delete(docId *metapb.DocID) bool {
//	deleteReq := pspb.BulkItemRequest{
//		OpType: pspb.OpType_DELETE,
//		Delete: &pspb.DeleteRequest{Id: *docId},
//	}
//	request := &pspb.BulkRequest{
//		ActionRequestHeader: partition.requestHeader,
//		Requests:            []pspb.BulkItemRequest{deleteReq},
//	}
//	ctx, cancel := partition.getContext()
//	defer cancel()
//	resp := partition.getSingleResponse(partition.getClient().BulkWrite(ctx, request)).Delete
//	if resp.Result != pspb.WriteResult_DELETED {
//		return false
//	}
//	return true
//}
//
//func (partition *Partition) getClient() pspb.ApiGrpcClient {
//	psClient, err := partition.psClient.GetGrpcClient(partition.leaderAddr)
//	if err != nil {
//		log.Warn("get ps client for %s failed", partition.leaderAddr)
//		panic(err)
//	}
//	return psClient.(pspb.ApiGrpcClient)
//}
//
//func (partition *Partition) getContext() (context.Context, context.CancelFunc) {
//	return context.WithTimeout(partition.parent.parent.context, rpcTimeoutDef)
//}
//
//func (partition *Partition) getSingleResponse(resp *pspb.BulkResponse, err error) *pspb.BulkItemResponse {
//	if err != nil {
//		log.Error("send bulk request failed: %s", err.Error())
//		panic(err)
//	}
//	if resp.Code != metapb.RESP_CODE_OK {
//		if resp.Code == metapb.PS_RESP_CODE_NO_LEADER || resp.Code == metapb.PS_RESP_CODE_NO_PARTITION {
//			partition.parent.Delete(partition.meta)
//		} else if resp.Code == metapb.PS_RESP_CODE_NOT_LEADER {
//			partition.leaderAddr = resp.Error.NotLeader.LeaderAddr
//		}
//		log.Error("bulk response failed(%d): %s", resp.Code, resp.Message)
//		panic(errors.New(resp.Message))
//	}
//	if len(resp.Responses) != 1 || resp.Responses[0].OpType != pspb.OpType_CREATE {
//		panic(errors.New("bad response for OpType_CREATE"))
//	}
//	return &resp.Responses[0]
//}
