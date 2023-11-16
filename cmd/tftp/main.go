package main

import (
	"fmt"
	"ncd/homework/tftp"
)

func main() {
	data := []byte{0, 4, 0, 4}
	op, _ := tftp.PeekOp(data)

	fmt.Printf("Received op: %s\n", op)

	switch op {
	case tftp.OpAck:
		var packet tftp.PacketAck
		packet.UnmarshalBinary(data)
		fmt.Printf("ACK - #%d\n", packet.BlockNum)
	default:
		fmt.Println("Unhandled operation")
	}

	// TODO implement the in-memory tftp server
}
