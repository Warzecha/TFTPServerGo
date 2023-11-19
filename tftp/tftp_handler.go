package tftp

import (
	"fmt"
	"net"
)

func TftpHandler(connection *net.UDPConn, quit chan struct{}) {

	buffer := make([]byte, 512)

	for {
		_, addr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Read error: %s \n", err)
		}

		// fmt.Print("-> ", string(buffer[0:n-1]))

		op, _ := PeekOp(buffer)

		fmt.Printf("Received op: %s\n", op)

		switch op {
		case OpAck:
			var packet PacketAck
			packet.UnmarshalBinary(buffer)
			fmt.Printf("ACK - #%d\n", packet.BlockNum)

		case OpWrite:
			var requestPacket PacketRequest
			requestPacket.UnmarshalBinary(buffer)
			// fmt.Printf("Write - #%d\n", dataPacket.Data[])

			fmt.Printf("Filename: %s\n", string(requestPacket.Filename))
			fmt.Printf("Mode: %s\n", string(requestPacket.Mode))

			ackPacket := PacketAck{
				Op:       OpAck,
				BlockNum: 0,
			}

			data, err := ackPacket.MarshalBinary()

			if err != nil {
				fmt.Printf("ACK marshalling error: %s \n", err)
			}

			_, err = connection.WriteToUDP(data, addr)

			if err != nil {
				fmt.Printf("ACK write error: %s \n", err)
			}

		case OpData:

		default:
			fmt.Println("Unhandled operation")
		}

	}

}
