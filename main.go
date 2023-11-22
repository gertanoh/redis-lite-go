package main

import (
	"fmt"
	"log"
	"net"
)

func main() {

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	fmt.Println("Listenning on port :6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go HandleConnection(conn)
	}
}
