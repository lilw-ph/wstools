package pingbase

import "sync"

type PingStatus uint8

const (
	Succeed PingStatus = 0
	Refused PingStatus = 1
	Timeout PingStatus = 2
)

type PingTask struct {
	Protocol  string
	DAddr     string
	SAddr     string
	SPort     uint16
	DPort     uint16
	Interval  uint16
	Timeout   uint32
	PkgCnt    uint32
	Index     uint32
	SendTime  int64
	Res       PingRes
	ResInfoCh chan *ResInfo
}

type PingRes struct {
	CnntStatus       PingStatus
	PortStatus       PingStatus
	SndCnt           uint32
	RcvCnt           uint32
	MinRelay         float64
	MaxRelay         float64
	TotalRelay       float64
	TotalSquareRelay float64
}

type ResInfo struct {
	TTL    int
	Bytes  uint32
	RcvCnt uint32
	Relay  float64
}

type ResData struct {
	LAddr string
	DAddr string
	SPort uint16
	DPort uint16
}

type Pinger interface {
	SendPkg(syncFlag *SyncFlag)
}

type SyncFlag struct {
	Wg *sync.WaitGroup
}

type Frecive func(string, uint8, map[string]*PingTask, *SyncFlag)
