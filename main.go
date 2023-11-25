package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ger/redis-lite-go/internal/handler"
)

func main() {

	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	fmt.Println("Listenning on port :6379")

	aof, err := handler.NewAof()
	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		go handler.HandleConnection(conn, aof)
	}
}
