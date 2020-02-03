package wsutil

import (
	"errors"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func InterfaceAddress(ifaceName string) string {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		panic(err)
	}

	addr, err := iface.Addrs()
	if err != nil {
		panic(err)
	}

	addrStr := strings.Split(addr[0].String(), "/")[0]

	return addrStr
}

func PortSplit(portRange *string) (uint16, uint16) {
	ports := strings.Split(*portRange, "-")
	minPort, err := strconv.ParseUint(ports[0], 10, 16)
	if err != nil {
		panic(err)
	}
	maxPort, err := strconv.ParseUint(ports[1], 10, 16)
	if err != nil {
		panic(err)
	}

	if minPort > maxPort {
		panic(errors.New("maxPort must greater than minPort"))
	}

	return uint16(minPort), uint16(maxPort)
}

func GetPort(strPort string) uint16 {
	port, err := strconv.ParseUint(strPort, 10, 16)
	if err != nil {
		panic(err)
	}

	return uint16(port)
}

func Random(min, max int) int {
	return rand.Intn(max-min) + min
}
