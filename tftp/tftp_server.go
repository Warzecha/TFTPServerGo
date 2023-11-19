package tftp

import (
	"fmt"
	"log"
	"net"
)

const (
	min_data_port int = 49152
	max_data_port int = 65535

	dummy_data_port int = 49154
)

type TftpServer struct {
	main_udp_server UdpServer

	secondary_udp_connection UdpServer
	// data_udp_connections map[string]

}

func (s *TftpServer) Start() {

	main_udp_server := UdpServer{
		Port:        6969,
		ThreadCount: 1,
		Handler:     s.ConnectionInitializationHandler,
	}

	main_udp_server.Start()

}

func (s *TftpServer) ConnectionInitializationHandler(connection *net.UDPConn) {

	buffer := make([]byte, 512)

	for {
		_, addr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Print("-> ", string(buffer[0:n-1]))

		op, _ := PeekOp(buffer)

		fmt.Printf("ConnectionInitializationHandler Received op: %s\n", op)

		switch op {
		case OpWrite:
			var requestPacket PacketRequest
			requestPacket.UnmarshalBinary(buffer)
			// fmt.Printf("Write - #%d\n", dataPacket.Data[])

			fmt.Printf("Received Write request for file: %s with mode: %s\n", requestPacket.Filename, requestPacket.Mode)

			ackPacket := PacketAck{
				Op:       OpAck,
				BlockNum: 0,
			}

			data, err := ackPacket.MarshalBinary()

			s.handleError(err)

			s.secondary_udp_connection = UdpServer{
				Port:        dummy_data_port,
				ThreadCount: 1,
				Handler:     s.DataHandler,
			}

			s.secondary_udp_connection.Start()

			data_conn := s.secondary_udp_connection.Connection

			_, err = data_conn.WriteToUDP(data, addr)

			s.handleError(err)
		case OpRead:
			var requestPacket PacketRequest
			requestPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received Read request for file: %s with mode: %s\n", requestPacket.Filename, requestPacket.Mode)

		default:
			fmt.Println("Unhandled operation")
		}

	}

}

func (s *TftpServer) DataHandler(connection *net.UDPConn) {

	buffer := make([]byte, 512)

	for {
		_, _, err := connection.ReadFromUDP(buffer)
		s.handleError(err)

		// fmt.Print("-> ", string(buffer[0:n-1]))

		op, _ := PeekOp(buffer)

		fmt.Printf("DataHandler Received op: %s\n", op)

		switch op {
		case OpData:
			var dataPacket PacketData
			dataPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received Data request for with BlockNum: %s\n", string(dataPacket.BlockNum))

		case OpAck:
			var ackPacket PacketAck
			ackPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received ACK\n")

		default:
			fmt.Printf("DataHandler Unhandled operation: %s \n", op)
		}

	}
}

func (s *TftpServer) handleError(err error) {
	log.Fatal(err)
}
