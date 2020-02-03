package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"wstools/util"
	"wstools/wsping/pingbase"
	"wstools/wsping/pingtcp"
)

var (
	localAddr  string
	remoteAddr string
	protocol   string
	interval   uint64
	port       uint64
	timeout    uint64
	count      uint64
	inFile     string
	outFile    string
)

func init() {

	regFlagParam()

}

func getLocalAddr() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, address := range addrs {
		// 检查ip是否为回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localAddr = ipnet.IP.String()
				break
			}
		}
	}
}

func regFlagParam() {
	flag.Uint64Var(&interval, "i", 1000, "interval ms")
	flag.StringVar(&protocol, "t", "auto", "protocol")
	flag.Uint64Var(&port, "p", 80, "port")
	flag.Uint64Var(&timeout, "T", 2000, "timeout ms")
	flag.Uint64Var(&count, "c", 10, "send pkg number")
	flag.StringVar(&inFile, "f", "", ".in file")
	flag.StringVar(&outFile, "o", "", ".out file")
	flag.StringVar(&localAddr, "l", "", "local addr")
	flag.StringVar(&remoteAddr, "r", "", "remote addr")
}

func initTasks() []*pingbase.PingTask {
	flag.Parse()
	if localAddr == "" {
		getLocalAddr()
	}

	tasks := make([]*pingbase.PingTask, 0)

	if inFile == "" {
		tasks = append(tasks, genTask())
	} else {
		initFileTasks(&tasks, inFile)
	}

	return tasks
}

func initFileTasks(tasks *[]*pingbase.PingTask, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	rd := bufio.NewReader(file)
	for {
		line, err := rd.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		(*tasks) = append((*tasks), parseTask(line))
	}
}

func genTask() *pingbase.PingTask {

	task := &pingbase.PingTask{
		Protocol:  protocol,
		DAddr:     remoteAddr,
		DPort:     uint16(port),
		Interval:  uint16(interval),
		SAddr:     localAddr,
		SPort:     uint16(wstoolutil.Random(10000, 65535)),
		Timeout:   uint32(timeout),
		PkgCnt:    uint32(count),
		ResInfoCh: make(chan *pingbase.ResInfo),
		Res: pingbase.PingRes{
			CnntStatus:       pingbase.Timeout,
			PortStatus:       pingbase.Timeout,
			SndCnt:           0,
			RcvCnt:           0,
			MinRelay:         0,
			MaxRelay:         0,
			TotalRelay:       0,
			TotalSquareRelay: 0,
		},
	}

	return task
}

func parseTask(cmd string) *pingbase.PingTask {

	cmd = strings.Replace(cmd, "\n", "", -1)

	flags := strings.Split(cmd, " ")
	os.Args = os.Args[:1]
	for _, v := range flags {
		os.Args = append(os.Args, v)
	}

	flag.Parse()

	return genTask()
}

func main() {

	taskList := initTasks()

	syncFlag := &pingbase.SyncFlag{
		Wg: new(sync.WaitGroup),
	}
	starFlag := make(chan bool, 1)
	var count int = len(taskList)

	concurrency := len(taskList)
	jobs := make(chan *pingbase.Pinger, concurrency)
	jobsMap := make(map[string]*pingbase.PingTask)

	rcvFMap := make(map[string]pingbase.Frecive)

	go func(jobs <-chan *pingbase.Pinger, starFlag chan bool) {
		for j := range jobs {
			syncFlag.Wg.Add(1)
			go (*j).SendPkg(syncFlag)
			starFlag <- true
		}
	}(jobs, starFlag)

	for i := 0; i < len(taskList); i++ {
		var ping pingbase.Pinger
		var key string
		var rcvFKey string
		var rcvF pingbase.Frecive

		task := taskList[i]

		key = task.Protocol + ":" + task.DAddr + ":" + strconv.FormatUint(uint64(task.DPort), 10)
		rcvFKey = task.Protocol + ":" + task.SAddr
		switch {
		case task.Protocol == "tcp":
			tcp4 := pingtcp.Tcp4{Task: task}
			ping = &tcp4
			rcvF = pingtcp.RcvPkg
		case task.Protocol == "udp":
		default:
			fmt.Println("Uknow protocol")
		}

		jobsMap[key] = task
		jobs <- &ping
		if _, ok := rcvFMap[rcvFKey]; !ok {
			rcvFMap[rcvFKey] = rcvF
		}

		fmt.Println(localAddr, key)
	}

	for k, v := range rcvFMap {
		if arr := strings.Split(k, ":"); len(arr) == 2 {
			v(arr[1], 10, jobsMap, syncFlag)
		}

	}

	var starNum int = 0
	for {
		select {
		case <-starFlag:
			starNum++
			if starNum == count {
				syncFlag.Wg.Wait()
				os.Exit(0)
			}
		}
	}

}
