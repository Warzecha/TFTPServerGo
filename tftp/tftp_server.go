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
)

type TftpServer struct {
	Port        int
	fileStorage FileStorage
	uploads     map[string]string
	downloads   map[string]DownloadMetadata
	quit        chan bool
}

func NewServer(port int) *TftpServer {
	return &TftpServer{
		Port:        port,
		uploads:     map[string]string{},
		downloads:   map[string]DownloadMetadata{},
		fileStorage: CreateEmptyMemoryStorage(),
	}
}

func (s *TftpServer) Start() error {
	fmt.Printf("Starting TFTP server on port %d \n", s.Port)

	udpAddress, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+strconv.Itoa(s.Port))

	if err != nil {
		fmt.Println("Error resolving server address.")
		return err
	}

	connection, err := net.ListenUDP("udp4", udpAddress)

	if err != nil {
		fmt.Printf("Error listening on address: %s \n", udpAddress)
		return err
	}

	defer connection.Close()

	s.quit = make(chan bool)

	buffer := make([]byte, 512)

	for {
		select {
		case <-s.quit:
			fmt.Printf("Terminating server...")
			return nil
		default:
			s.acceptReqest(connection, buffer)
		}
	}
}

func (s *TftpServer) Terminate() {
	s.quit <- true
}

func (s *TftpServer) connectionInitializationHandler(connection *net.UDPConn, quit chan bool) {
	buffer := make([]byte, 512)

	for {
		select {
		case <-quit:
			fmt.Println("Exiting connectionInitializationHandler...")
			return
		default:
			s.acceptReqest(connection, buffer)
		}
	}
}

func (s *TftpServer) acceptReqest(connection *net.UDPConn, buffer []byte) {

	_, addr, err := connection.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal(err)
	}

	op, _ := PeekOp(buffer)

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

		if err != nil {
			fmt.Printf("Error when resolving address for data port: %s. \n", data_port_str)
			s.sendError(connection, addr, ErrNotDefined, "Unknown error occurred.")
			break
		}

		data_connection, err := net.DialUDP("udp4", data_udp_address, addr)

		if err != nil {
			fmt.Printf("Error when opening data connection on port: %s. \n", data_port_str)
			s.sendError(connection, addr, ErrNotDefined, "Unknown error occurred.")
			break
		}

		go s.dataWriteHandler(data_connection, data_port_str)

		s.sendAck(data_connection, 0)

	case OpRead:
		var requestPacket PacketRequest
		requestPacket.UnmarshalBinary(buffer)

		fmt.Printf("Received Read request for file: %s with mode: %s\n", requestPacket.Filename, requestPacket.Mode)

		if requestPacket.Mode != "octet" {
			s.sendError(connection, addr, ErrIllegal, "Only octet mode is supported")
			break
		}

		if _, exists := s.fileStorage.GetFileMetadata(requestPacket.Filename); !exists {
			s.sendError(connection, addr, ErrFileNotFound, fmt.Sprintf("File with name '%s' does not exist.", requestPacket.Filename))
			break
		}

		data_port := s.selectRandomPort()
		data_port_str := strconv.Itoa(data_port)

		fmt.Printf("Will use data port: %s \n", data_port_str)

		data_udp_address, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+data_port_str)

		if err != nil {
			fmt.Printf("Error when resolving address for data port: %s. \n", data_port_str)
			s.sendError(connection, addr, ErrNotDefined, "Unknown error occurred.")
			break
		}

		s.downloads[data_port_str] = DownloadMetadata{
			Filename:     requestPacket.Filename,
			LastBlockNum: 0,
		}

		data_connection, err := net.DialUDP("udp4", data_udp_address, addr)

		if err != nil {
			fmt.Printf("Error when opening data connection on port: %s. \n", data_port_str)
			s.sendError(connection, addr, ErrNotDefined, "Unknown error occurred.")
			break
		}

		go s.dataReadHandler(data_connection, data_port_str)

		s.readAndSendFile(data_connection, requestPacket.Filename, 0)
	case OpAck:
		fmt.Printf("Received ACK")
	default:
		fmt.Println("Unhandled operation")
	}

}

func (s *TftpServer) dataWriteHandler(connection *net.UDPConn, port string) {
	for {
		buffer := make([]byte, 516)
		n, _, err := connection.ReadFromUDP(buffer)
		s.logErrorIfExists(err)

		op, _ := PeekOp(buffer)

		switch op {
		case OpData:
			var dataPacket PacketData
			dataPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received Data request for with BlockNum: %d on port %s\n", dataPacket.BlockNum, port)

			if dataPacket.BlockNum == 1 {
				s.fileStorage.StartNewUpload(s.uploads[port])
			}

			s.fileStorage.AppendData(s.uploads[port], int(dataPacket.BlockNum), dataPacket.Data[0:n-4])

			is_complete := n < 516
			if is_complete {
				s.fileStorage.CompleteUpload(s.uploads[port])
				delete(s.uploads, port)
				is_complete = true
			}

			s.sendAck(connection, dataPacket.BlockNum)

			if is_complete {
				defer connection.Close()
			}

		case OpAck:
			var ackPacket PacketAck
			ackPacket.UnmarshalBinary(buffer)

			fmt.Printf("Received ACK\n")

		default:
			fmt.Printf("DataHandler Unhandled operation: %s \n", op)
		}

	}
}

func (s *TftpServer) dataReadHandler(connection *net.UDPConn, port string) {
	buffer := make([]byte, 512)

	for {
		_, _, err := connection.ReadFromUDP(buffer)
		s.logErrorIfExists(err)

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
	s.logErrorIfExists(err)

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

func (s *TftpServer) sendError(connection *net.UDPConn, addr *net.UDPAddr, errCode ErrorCode, msg string) {
	errPacket := PacketError{
		Op:    OpError,
		Error: errCode,
		Msg:   msg,
	}

	err_data, err := errPacket.MarshalBinary()

	if err != nil {
		fmt.Printf("Error marshalling error: %s \n", err)
	}

	_, err = connection.WriteToUDP(err_data, addr)

	if err != nil {
		fmt.Printf("Error packet write error: %s \n", err)
	}
}

func (s *TftpServer) logErrorIfExists(err error) {
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
