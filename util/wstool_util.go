package wstoolutil

import (
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func CheckError(err error) {
	if err != nil {
		log.Println(err)
	}
}

func Ipstr2Bytes(addr string) [4]byte {
	s := strings.Split(addr, ".")
	b0, _ := strconv.Atoi(s[0])
	b1, _ := strconv.Atoi(s[1])
	b2, _ := strconv.Atoi(s[2])
	b3, _ := strconv.Atoi(s[3])

	return [4]byte{byte(b0), byte(b1), byte(b2), byte(b3)}
}

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
