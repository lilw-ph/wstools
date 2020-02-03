package main

import (
	"flag"
	"fmt"

	"ping.tool/tcp"
	"ping.tool/util"
)

func main() {
	ifaceName := flag.String("i", "eth0", "Specify network")
	remote := flag.String("r", "", "remote address")
	port := flag.String("p", "80", "port range: -p 1-1024")
	flag.Parse()

	laddr := wsutil.InterfaceAddress(*ifaceName)
	raddr := *remote
	sport := uint16(wsutil.Random(10000, 65535))
	dport := wsutil.GetPort(*port)
	fmt.Println(laddr, raddr)

	wstcpping.SendSyn(laddr, raddr, sport, dport)
	wstcpping.RecvSynAck(laddr, raddr)
}
