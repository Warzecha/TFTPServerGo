package tftp

import (
	"log"
	"net"
	"strconv"
	"time"
)

type UdpServer struct {
	Port        int
	Handler     func(connection *net.UDPConn)
	ThreadCount int
	Timeout     time.Duration

	Connection *net.UDPConn
}

func (s *UdpServer) Start() {
	udpAddress, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+strconv.Itoa(s.Port))
	if err != nil {
		log.Fatal(err)
	}

	connection, err := net.ListenUDP("udp4", udpAddress)
	if err != nil {
		log.Fatal(err)
	}

	defer connection.Close()

	s.Connection = connection

	s.Handler(connection)

}
