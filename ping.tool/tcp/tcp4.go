package wstcpping

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

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

func checkError(err error) {
	if err != nil {
		log.Println(err)
	}
}

func mySum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(uint16(data[i])<<8 | uint16(data[i+1]))
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum = sum + (sum >> 16)

	return uint16(sum)
}

func ipstr2Bytes(addr string) [4]byte {
	s := strings.Split(addr, ".")
	b0, _ := strconv.Atoi(s[0])
	b1, _ := strconv.Atoi(s[1])
	b2, _ := strconv.Atoi(s[2])
	b3, _ := strconv.Atoi(s[3])

	return [4]byte{byte(b0), byte(b1), byte(b2), byte(b3)}
}

func CheckSum(data []byte, src, dst [4]byte) uint16 {
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

func SendSyn(laddr, raddr string, sport, dport uint16) {
	conn, err := net.Dial("ip4:tcp", raddr)
	checkError(err)
	defer conn.Close()
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
		SrcPort:       sport,
		DstPort:       dport,
		SeqNum:        rand.Uint32(),
		AckNum:        0,
		Flags:         0x8002,
		Window:        8192,
		ChkSum:        0,
		UrgentPointer: 0,
	}

	buff := new(bytes.Buffer)

	err = binary.Write(buff, binary.BigEndian, tcpH)
	checkError(err)
	for i := range op {
		binary.Write(buff, binary.BigEndian, op[i].Kind)
		binary.Write(buff, binary.BigEndian, op[i].Length)
		binary.Write(buff, binary.BigEndian, op[i].Data)
	}
	binary.Write(buff, binary.BigEndian, [6]byte{})
	data := buff.Bytes()
	checkSum := CheckSum(data, ipstr2Bytes(laddr), ipstr2Bytes(raddr))
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

	_, err = conn.Write(data)
	checkError(err)
}

func RecvSynAck(laddr, raddr string) error {
	listenAddr, err := net.ResolveIPAddr("ip4", laddr)
	checkError(err)
	conn, err := net.ListenIP("ip4:tcp", listenAddr)
	defer conn.Close()
	checkError(err)
	for {
		buff := make([]byte, 1024)
		_, addr, err := conn.ReadFrom(buff)
		if err != nil {
			continue
		}

		if addr.String() != raddr || buff[13] != 0x12 {
			continue
		}

		var port uint16
		binary.Read(bytes.NewBuffer(buff), binary.BigEndian, &port)

		//fmt.Println("port: ", port, " opened")
		fmt.Printf("%s:%d opened", conn.LocalAddr().String(), port)
	}
}
