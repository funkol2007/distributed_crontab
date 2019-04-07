package logid

import (
	"helper/common"
	"sync/atomic"
)

var LogId int64

func GenerRateLogId() int64 {
	return atomic.AddInt64((*int64)(&LogId), 1)
}

func init() {
	LogId = common.NowInNs()
}
