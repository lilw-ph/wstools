package pingtcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
	"wstools/util"
	"wstools/wsping/pingbase"

	"golang.org/x/net/ipv4"
)

type Tcp4 struct {
	Task *pingbase.PingTask
}

// tcp header
type TCPHeader struct {
	SrcPort       uint16
	DstPort       uint16
	SeqNum        uint32
	AckNum        uint32
	Flags         uint16
	Window        uint16
	ChkSum        uint16
	UrgentPointer uint16
}

// tcp option
type TCPOption struct {
	Kind   uint8
	Length uint8
	Data   []byte
}

type tcpipPkg struct {
	ipH  *ipv4.Header
	tcpH []byte
}

var tcpLodk sync.Mutex

func mySum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(uint16(data[i])<<8 | uint16(data[i+1]))
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum = sum + (sum >> 16)

	return uint16(sum)
}

func checkSum(data []byte, src, dst [4]byte) uint16 {
	pseudoHeader := []byte{
		src[0], src[1], src[2], src[3],
		dst[0], dst[1], dst[2], dst[3],
		0,
		6,
		0,
		byte(len(data)),
	}

	totalLength := len(pseudoHeader) + len(data)
	if totalLength%2 != 0 {
		totalLength++
	}

	d := make([]byte, 0, totalLength)
	d = append(d, pseudoHeader...)
	d = append(d, data...)

	return ^mySum(d)
}

func (t *Tcp4) SendPkg(syncFlag *pingbase.SyncFlag) {
	defer syncFlag.Wg.Done()
	defer fmt.Println("SendPkg done")

	var flag bool
	var isRcv bool
	task := t.Task
	fmt.Printf("[%d] %s, %s:%d -> %s:%d, %d[%d * %d]\n",
		task.Index, task.Protocol, task.SAddr, task.SPort, task.DAddr, task.DPort,
		task.Timeout, task.Interval, task.PkgCnt)

	conn, err := net.DialTimeout("ip4:tcp", t.Task.DAddr, time.Duration(t.Task.Timeout)*time.Millisecond)
	wstoolutil.CheckError(err)
	defer conn.Close()

	for c := uint32(0); c < task.PkgCnt; c++ {
		flag = true
		isRcv = false
		/*
			conn, err := net.DialTimeout("ip4:tcp", t.Task.DAddr, time.Duration(t.Task.Timeout)*time.Millisecond)
			wstoolutil.CheckError(err)
			defer conn.Close()
		*/
		op := []TCPOption{
			TCPOption{
				Kind:   2,
				Length: 4,
				Data:   []byte{0x05, 0xb4},
			},
			TCPOption{
				Kind: 0,
			},
		}

		tcpH := TCPHeader{
			SrcPort:       t.Task.SPort,
			DstPort:       t.Task.DPort,
			SeqNum:        t.Task.Res.RcvCnt,
			AckNum:        0,
			Flags:         0x8002,
			Window:        8192,
			ChkSum:        0,
			UrgentPointer: 0,
		}

		buff := new(bytes.Buffer)

		err = binary.Write(buff, binary.BigEndian, tcpH)
		wstoolutil.CheckError(err)
		for i := range op {
			binary.Write(buff, binary.BigEndian, op[i].Kind)
			binary.Write(buff, binary.BigEndian, op[i].Length)
			binary.Write(buff, binary.BigEndian, op[i].Data)
		}
		binary.Write(buff, binary.BigEndian, [6]byte{})
		data := buff.Bytes()
		checkSum := checkSum(data, wstoolutil.Ipstr2Bytes(t.Task.SAddr), wstoolutil.Ipstr2Bytes(t.Task.DAddr))
		tcpH.ChkSum = checkSum

		buff = new(bytes.Buffer)
		binary.Write(buff, binary.BigEndian, tcpH)
		for i := range op {
			binary.Write(buff, binary.BigEndian, op[i].Kind)
			binary.Write(buff, binary.BigEndian, op[i].Length)
			binary.Write(buff, binary.BigEndian, op[i].Data)
		}
		binary.Write(buff, binary.BigEndian, [6]byte{})
		data = buff.Bytes()

		tcpLodk.Lock()
		task.SendTime = time.Now().UnixNano()
		tcpLodk.Unlock()

		_, err = conn.Write(data)
		wstoolutil.CheckError(err)
		if err == nil {
			t.Task.Res.SndCnt++
		}

		for flag {
			select {
			case resInfo := <-task.ResInfoCh:
				fmt.Printf("from %s: seq = %d ttl=%d time=%.2fms\n",
					task.DAddr, resInfo.RcvCnt, resInfo.TTL, resInfo.Relay)
				isRcv = true
			case <-time.After(time.Duration(time.Millisecond * time.Duration(task.Interval))):
				flag = false
				if !isRcv {
					fmt.Println("timeout ....")
				}
			}
		}
	}

	var (
		loss  float64 = 0
		min   float64
		max   float64
		avg   float64 = 0
		tsum  float64
		tsum2 float64
		mdev  float64 = 0
	)
	if t.Task.Res.SndCnt != 0 {
		loss = float64(t.Task.Res.SndCnt-t.Task.Res.RcvCnt) / float64(t.Task.Res.SndCnt) * 100
	}
	if t.Task.Res.RcvCnt != 0 {
		avg = t.Task.Res.TotalRelay / float64(t.Task.Res.RcvCnt)
		tsum = t.Task.Res.TotalRelay / float64(t.Task.Res.RcvCnt)
		tsum2 = t.Task.Res.TotalSquareRelay / float64(t.Task.Res.RcvCnt)
		mdev = math.Sqrt(tsum2 - (tsum * tsum))
	}
	min = t.Task.Res.MinRelay
	max = t.Task.Res.MaxRelay

	fmt.Printf("%s, %d times, sent/recv/loss = %d/%d/%.2f%%, min/avg/max/mdev = %.4f/%.4f/%.4f/%.4f ms, conn_status/port_status/port = %d/%d/%d\n",
		t.Task.DAddr, t.Task.PkgCnt,
		t.Task.Res.SndCnt, t.Task.Res.RcvCnt, loss,
		min, avg, max, mdev,
		t.Task.Res.CnntStatus, t.Task.Res.PortStatus, t.Task.DPort)
}

func RcvPkg(saddr string, rn uint8, taskMap map[string]*pingbase.PingTask, syncFlag *pingbase.SyncFlag) {
	fmt.Printf("RcvPkg %s\n", saddr)
	ch := make(chan *tcpipPkg)
	go rcvpkg(saddr, rn, ch, syncFlag)
	for i := uint8(0); i < rn; i++ {
		go processPkg(ch, taskMap, syncFlag)
	}
}

func processPkg(ch chan *tcpipPkg, taskMap map[string]*pingbase.PingTask, syncFlag *pingbase.SyncFlag) {

	var (
		relay   float64
		pkgFlag uint8
		ack     bool
		rst     bool
		portSta pingbase.PingStatus
	)

	for {
		pkg := <-ch

		ipHdr := pkg.ipH
		tcpHdr := new(TCPHeader)
		binary.Read(bytes.NewBuffer(pkg.tcpH), binary.BigEndian, tcpHdr)

		pkgFlag = uint8(tcpHdr.Flags & 0x3F)
		ack = (pkgFlag&0x02 != 0) && (pkgFlag&0x10 != 0)
		rst = (pkgFlag&0x04 != 0)
		if !(ack || rst) {
			continue
		}
		if ack {
			portSta = pingbase.Succeed
		} else {
			portSta = pingbase.Refused
		}

		key := "tcp" + ":" + ipHdr.Src.String() + ":" + strconv.FormatUint(uint64(tcpHdr.SrcPort), 10)
		task, ok := taskMap[key]
		if !ok {
			continue
		}

		tcpLodk.Lock()
		relay = float64(time.Now().UnixNano()-task.SendTime) / 1000000.0
		resInfo := &pingbase.ResInfo{
			TTL:    ipHdr.TTL,
			Bytes:  0,
			RcvCnt: tcpHdr.AckNum,
			Relay:  relay,
		}
		if tcpHdr.AckNum == 1 {
			task.Res.MinRelay = relay
			task.Res.MaxRelay = relay
		}
		if task.Res.MinRelay > relay {
			task.Res.MinRelay = relay
		}
		if task.Res.MaxRelay < relay {
			task.Res.MaxRelay = relay
		}
		task.Res.TotalRelay += relay
		task.Res.TotalSquareRelay += relay * relay
		task.Res.RcvCnt = tcpHdr.AckNum

		if task.Res.PortStatus > portSta {
			task.Res.PortStatus = portSta
		}
		task.Res.CnntStatus = pingbase.Succeed

		tcpLodk.Unlock()

		task.ResInfoCh <- resInfo
	}
}

func rcvpkg(saddr string, rn uint8, ch chan *tcpipPkg, syncFlag *pingbase.SyncFlag) {
	listenAddr, err := net.ResolveIPAddr("ip4", saddr)
	wstoolutil.CheckError(err)
	conn, err := net.ListenIP("ip4:tcp", listenAddr)
	ipconn, _ := ipv4.NewRawConn(conn)
	defer conn.Close()
	wstoolutil.CheckError(err)
	for {
		buf := make([]byte, 1480)
		hdr, payload, _, _ := ipconn.ReadFrom(buf)

		rcvData := &tcpipPkg{
			ipH:  hdr,
			tcpH: payload,
		}

		ch <- rcvData
	}
}
