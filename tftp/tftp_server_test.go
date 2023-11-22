package tftp

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	min_server_port int = 49152
	max_server_port int = 65535
)

func TestReadFileDoesNotExist(t *testing.T) {
	tftp_server, mock_file_storage, server_port, client_port := getTestResources(t)
	mock_file_storage.EXPECT().GetFileMetadata("non-existing-file").Return(FileMetadata{}, false)

	go func() {
		err := tftp_server.Start()
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	conn := createClientServerConnection(t, client_port, server_port)
	defer conn.Close()

	sendReadRequest(t, conn, "non-existing-file", "octet")
	assertReceivedError(t, conn, ErrFileNotFound)
}

func TestReadInvalidMode(t *testing.T) {
	tftp_server, _, server_port, client_port := getTestResources(t)

	go func() {
		err := tftp_server.Start()
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	conn := createClientServerConnection(t, client_port, server_port)
	defer conn.Close()

	sendReadRequest(t, conn, "any_file", "netascii")
	assertReceivedError(t, conn, ErrIllegal)
}

func TestReadFileExistsAndUnderOneBlock(t *testing.T) {
	tftp_server, mock_file_storage, server_port, client_port := getTestResources(t)
	file_metadata := FileMetadata{
		Filename:   "existing-file",
		IsComplete: true,
	}

	fmt.Printf("TestReadFileExistsAndUnderOneBlock server_port: %d, client_port: %d \n", server_port, client_port)

	mock_file_storage.EXPECT().GetFileMetadata("existing-file").Return(file_metadata, true)
	mock_file_storage.EXPECT().ReadFileBytes("existing-file", 0, 512).Return([]byte("Hello World"))

	go func() {
		err := tftp_server.Start()
		assert.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)

	conn := createClientServerConnection(t, client_port, server_port)

	sendReadRequest(t, conn, "existing-file", "octet")
	conn.Close()
	// assertReceivedData(t, client_port, []byte("Hello World"))
}

func getTestResources(t *testing.T) (TftpServer, *MockFileStorage, int, int) {
	server_port := selectRandomPort()
	client_port := selectRandomPort()
	mock_file_storage := NewMockFileStorage(t)

	tftp_server := TftpServer{
		Port:        server_port,
		fileStorage: mock_file_storage,
		uploads:     map[string]string{},
		downloads:   map[string]DownloadMetadata{},
	}

	return tftp_server, mock_file_storage, server_port, client_port
}

func createClientServerConnection(t *testing.T, client_port int, server_port int) *net.UDPConn {
	server_addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+strconv.Itoa(server_port))
	assert.NoError(t, err, "Failed to resolve server address.")

	client_addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+strconv.Itoa(client_port))
	assert.NoError(t, err, "Failed to resolve client address.")

	conn, err := net.DialUDP("udp4", client_addr, server_addr)

	assert.NoError(t, err, "Failed to connect to server.")
	return conn
}

func sendReadRequest(t *testing.T, conn *net.UDPConn, filename string, mode string) {
	read_packet := PacketRequest{
		Op:       OpRead,
		Filename: filename,
		Mode:     mode,
	}

	data, err := read_packet.MarshalBinary()
	assert.NoError(t, err)
	conn.Write(data)
}

func assertReceivedError(t *testing.T, conn *net.UDPConn, expectedErrorCode ErrorCode) {
	buffer := make([]byte, 512)

	conn.ReadFromUDP(buffer)

	op, err := PeekOp(buffer)
	assert.NoError(t, err)
	assert.Equal(t, OpError, op, "Expected to receive error packet.")

	var errorPacket PacketError
	errorPacket.UnmarshalBinary(buffer)

	assert.Equal(t, expectedErrorCode, errorPacket.Error, "Expected error code does not match.")
}

func assertReceivedData(t *testing.T, client_port int, expectedContent []byte) {
	client_addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+strconv.Itoa(client_port))
	assert.NoError(t, err)

	conn, err := net.ListenUDP("udp4", client_addr)
	assert.NoError(t, err)

	defer conn.Close()

	buffer := make([]byte, 512)

	conn.ReadFromUDP(buffer)

	op, err := PeekOp(buffer)
	assert.NoError(t, err)
	assert.Equal(t, OpData, op, "Expected to receive data packet.")

	var dataPacket PacketData
	dataPacket.UnmarshalBinary(buffer)

	assert.Equal(t, expectedContent, dataPacket.Data, "Expected data content does not match.")
}

func selectRandomPort() int {
	return rand.Intn(max_data_port-min_data_port) + min_data_port
}
