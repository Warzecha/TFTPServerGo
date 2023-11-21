package tftp

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
)

const (
	min_data_port int = 49152
	max_data_port int = 65535

	dummy_data_port int = 49155
)

type TftpServer struct {
	main_udp_server UdpServer

	secondary_udp_connection UdpServer

	fileStorage MemoryFileStorage
	uploads     map[string]string
	downloads   map[string]DownloadMetadata
}

func (s *TftpServer) Start() {

	main_udp_server := UdpServer{
		Port:        6969,
		ThreadCount: 1,
		Handler:     s.ConnectionInitializationHandler,
	}

	s.uploads = map[string]string{}
	s.downloads = map[string]DownloadMetadata{}

	s.fileStorage = MemoryFileStorage{
		files: map[string]*FileMetadata{
			"test2": &FileMetadata{
				Filename:   "test2",
				IsComplete: true,
			},
		},
		fileContents: map[string][]byte{
			"test2": []byte("Hello World"),
		},
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

		op, _ := PeekOp(buffer)

		fmt.Printf("ConnectionInitializationHandler Received op: %s\n", op)

		switch op {
		case OpWrite:
			var requestPacket PacketRequest
			requestPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received Write request for file: %s with mode: %s\n", requestPacket.Filename, requestPacket.Mode)

			data_port := s.selectRandomPort()
			data_port_str := strconv.Itoa(data_port)

			fmt.Printf("Will use data port: %s \n", data_port_str)

			data_udp_address, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+data_port_str)
			s.uploads[data_port_str] = requestPacket.Filename

			s.handleError(err)

			data_connection, err := net.DialUDP("udp4", data_udp_address, addr)
			s.handleError(err)

			defer data_connection.Close()

			go s.DataWriteHandler(data_connection, data_port_str)

			s.sendAck(data_connection, 0)

		case OpRead:
			var requestPacket PacketRequest
			requestPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received Read request for file: %s with mode: %s\n", requestPacket.Filename, requestPacket.Mode)

			// TODO Validate if the file exists and mode is correct

			data_port := s.selectRandomPort()
			data_port_str := strconv.Itoa(data_port)

			fmt.Printf("Will use data port: %s \n", data_port_str)

			data_udp_address, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+data_port_str)
			s.handleError(err)

			s.downloads[data_port_str] = DownloadMetadata{
				Filename:     requestPacket.Filename,
				LastBlockNum: 0,
			}

			data_connection, err := net.DialUDP("udp4", data_udp_address, addr)
			s.handleError(err)

			defer data_connection.Close()

			go s.DataReadHandler(data_connection, data_port_str)

			s.readAndSendFile(data_connection, requestPacket.Filename, 0)

		default:
			fmt.Println("Unhandled operation")
		}

	}

}

func (s *TftpServer) DataWriteHandler(connection *net.UDPConn, port string) {
	buffer := make([]byte, 512)

	for {
		_, _, err := connection.ReadFromUDP(buffer)
		s.handleError(err)

		op, _ := PeekOp(buffer)

		switch op {
		case OpData:
			var dataPacket PacketData
			dataPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received Data request for with BlockNum: %d on port %s\n", dataPacket.BlockNum, port)
			if dataPacket.BlockNum == 1 {
				s.fileStorage.StartNewUpload(s.uploads[port])
			}

			s.fileStorage.AppendData(s.uploads[port], int(dataPacket.BlockNum), dataPacket.Data)

			if len(dataPacket.Data) < 512 {
				s.fileStorage.CompleteUpload(s.uploads[port])
				delete(s.uploads, port)
			}

			s.sendAck(connection, dataPacket.BlockNum)

		case OpAck:
			var ackPacket PacketAck
			ackPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received ACK\n")

		default:
			fmt.Printf("DataHandler Unhandled operation: %s \n", op)
		}

	}
}

func (s *TftpServer) DataReadHandler(connection *net.UDPConn, port string) {
	buffer := make([]byte, 512)

	for {
		_, _, err := connection.ReadFromUDP(buffer)
		s.handleError(err)

		op, _ := PeekOp(buffer)

		switch op {
		case OpAck:
			var ackPacket PacketAck
			ackPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received ACK for with BlockNum: %d on port %s\n", ackPacket.BlockNum, port)

			s.readAndSendFile(connection, s.downloads[port].Filename, ackPacket.BlockNum)
		default:
			fmt.Printf("DataHandler Unhandled operation: %s \n", op)
		}

	}
}

func (s *TftpServer) readAndSendFile(connection *net.UDPConn, filename string, prevBlockNum uint16) {
	bytesStart := int(prevBlockNum * 512)
	bytesEnd := bytesStart + 512

	fileBlock := s.fileStorage.ReadFileBytes(filename, bytesStart, bytesEnd)

	dataPacket := PacketData{
		Op:       OpData,
		BlockNum: uint16(prevBlockNum + 1),
		Data:     fileBlock,
	}

	data_data, err := dataPacket.MarshalBinary()
	s.handleError(err)

	connection.Write(data_data)
}

func (s *TftpServer) sendAck(connection *net.UDPConn, blockNum uint16) {
	ackPacket := PacketAck{
		Op:       OpAck,
		BlockNum: blockNum,
	}

	ack_data, err := ackPacket.MarshalBinary()

	if err != nil {
		fmt.Printf("ACK marshalling error: %s \n", err)
	}

	_, err = connection.Write(ack_data)

	if err != nil {
		fmt.Printf("ACK write error: %s \n", err)
	}
}

func (s *TftpServer) handleError(err error) {
	if err != nil {
		fmt.Printf("Error occured: %s \n", err)
	}
}

func (s *TftpServer) handleCriticalError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (s *TftpServer) selectRandomPort() int {
	return rand.Intn(max_data_port-min_data_port) + min_data_port
}
