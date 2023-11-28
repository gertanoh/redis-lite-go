package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ger/redis-lite-go/internal/handler"
)

var (
	buildTime string
	version   string
)

func main() {

	displayVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Build time:\t%s\n", buildTime)
		os.Exit(0)
	}
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
