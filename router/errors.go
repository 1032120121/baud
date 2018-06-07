package router

import "errors"

var (
	ErrSuc           = errors.New("success")
	ErrInternalError = errors.New("internal error")
	ErrSysBusy       = errors.New("system busy")
	ErrUriError      = errors.New("uri error")
	ErrParamError    = errors.New("param error")
	ErrInvalidCfg    = errors.New("config error")
	ErrNotMSLeader   = errors.New("the master node is not a leader")
	ErrNoMSLeader    = errors.New("the master cluster have no a leader")

	ErrDupZone            = errors.New("duplicated zone")
	ErrDupDb              = errors.New("duplicated database")
	ErrDbNotExists        = errors.New("db not exists")
	ErrDupSpace           = errors.New("duplicated space")
	ErrSpaceNotExists     = errors.New("space not exists")
	ErrPartitionNotExists = errors.New("partition not exists")
	ErrPSNotExists        = errors.New("partition server is not exists")
	ErrGenIdFailed        = errors.New("generate id is failed")
	ErrLocalDbOpsFailed   = errors.New("local storage db operation error")
	ErrUnknownRaftCmdType = errors.New("unknown raft command type")
	ErrRouteNotFound      = errors.New("route not found")
)


// http response error code and error message definitions
const (
	ERRCODE_SUCCESS = iota
	ERRCODE_INTERNAL_ERROR
	ERRCODE_SYSBUSY
	ERRCODE_URI_ERROR
	ERRCODE_PARAM_ERROR
	ERRCODE_INVALID_CFG
	ERRCODE_NOT_MSLEADER
	ERRCODE_NO_MSLEADER

	ERRCODE_DUP_DB
	ERRCODE_DB_NOTEXISTS
	ERRCODE_DUP_SPACE
	ERRCODE_SPACE_NOTEXISTS
	ERRCODE_PS_NOTEXISTS
)

var Err2CodeMap = map[error]int32{
	ErrSuc:           ERRCODE_SUCCESS,
	ErrInternalError: ERRCODE_INTERNAL_ERROR,
	ErrSysBusy:       ERRCODE_SYSBUSY,
	ErrUriError:	  ERRCODE_URI_ERROR,
	ErrParamError:    ERRCODE_PARAM_ERROR,
	ErrInvalidCfg:    ERRCODE_INVALID_CFG,
	ErrNotMSLeader:   ERRCODE_NOT_MSLEADER,
	ErrNoMSLeader:    ERRCODE_NO_MSLEADER,

	ErrDupDb:          ERRCODE_DUP_DB,
	ErrDbNotExists:    ERRCODE_DB_NOTEXISTS,
	ErrDupSpace:       ERRCODE_DUP_SPACE,
	ErrSpaceNotExists: ERRCODE_SPACE_NOTEXISTS,
	ErrPSNotExists:    ERRCODE_PS_NOTEXISTS,
}
