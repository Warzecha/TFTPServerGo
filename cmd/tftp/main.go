package main

import (
	"ncd/homework/tftp"
)

const (
	ADDRESS string = "127.0.0.1:6969"
)

func main() {

	udp_server := tftp.TftpServer{}

	udp_server.Start()
}
