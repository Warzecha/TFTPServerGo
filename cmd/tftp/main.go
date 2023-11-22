package main

import (
	"fmt"
	"ncd/homework/tftp"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Required argument port.")
		return
	}

	port, err := strconv.Atoi(os.Args[1])

	if err != nil {
		fmt.Printf("Provided invalid port: %s \n", os.Args[1])
		return
	}

	tftp_server := tftp.NewServer(port)
	tftp_server.Start()
}
