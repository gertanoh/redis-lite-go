package main

import (
	"log"
	"net"
	"strings"
)

var handlers = map[string]func([]Payload) Payload{
	"PING":    ping,
	"COMMAND": command,
	"ECHO":    echo,
}

func HandleConnection(conn net.Conn) {

	respReader := NewRespReader(conn)
	cmd, err := respReader.Read()

	if err != nil {
		log.Println(err)
	}

	p := Payload{}
	if cmd.DataType != string(ARRAY) {
		p.DataType = string(ERROR)
		p.Str = "Expected array of bulk strings"
		return
	}

	if len(cmd.Array) == 0 {
		p.DataType = string(ERROR)
		p.Str = "Null array command"
		return
	}

	// first bulk string is the command
	request := strings.ToUpper(cmd.Array[0].Bulk)
	params := cmd.Array[1:]

	var response Payload
	if _, ok := handlers[request]; ok {
		response = handlers[request](params)
	} else {
		response.DataType = string(ERROR)
		response.Str = "Unknown command"
	}
	writer := NewRespWriter(conn)
	err = writer.Write(&response)
	if err != nil {
		log.Println(err)
	}

	conn.Close()
}

func ping(p []Payload) Payload {
	if len(p) == 0 {
		return Payload{DataType: string(STRING), Str: "PONG"}
	}
	return Payload{DataType: string(STRING), Str: p[0].Bulk}
}

func echo(p []Payload) Payload {
	if len(p) != 1 {
		return Payload{DataType: string(ERROR), Str: "Missing arguments for command"}
	}
	return Payload{DataType: string(STRING), Str: p[0].Bulk}
}

func command(p []Payload) Payload {
	return Payload{DataType: string(ARRAY), Array: []Payload{
		Payload{DataType: string(BULKSTRING), Bulk: "ECHO"},
		Payload{DataType: string(BULKSTRING), Bulk: "COMMAND"},
		Payload{DataType: string(BULKSTRING), Bulk: "PING"},
	}}
}
