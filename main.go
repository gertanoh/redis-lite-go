package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	fmt.Println("Listenning on port :6379")

	aof, err := NewAof()
	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		go HandleConnection(conn, aof)
	}
}
