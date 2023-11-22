package main

import (
	"log"
	"net"
	"strings"
	"time"
	"fmt"
	"strconv"
)

var handlers = map[string]func([]Payload) Payload{
	"PING":    ping,
	"COMMAND": command,
	"ECHO":    echo,
	"SET":     set,
	"GET":     get,
}


type stringValue struct {
	value string
	expire time.Time
}

var stringMap = map[string]stringValue{}

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

func set(p []Payload) Payload {

	key := p[0].Bulk
	value := p[1].Bulk
	var expire time.Time

	// Only handling EX options
	if len(p) >= 4 {
		ex_cmd := p[2].Bulk
		ex_val := p[3].Bulk
		if ex_cmd == "EX" {
			// Set expiration time
			expireInSecs, err := strconv.Atoi(ex_val)
			if err != nil {
				expire = time.Now().Add(time.Duration(expireInSecs) * time.Second)
			}
		}
	}


	stringMap[key] = stringValue{value, expire}
	return Payload{DataType: string(STRING), Str: "OK"}
}

func get(p []Payload) Payload {
	key := p[0].Bulk
	if _, ok := stringMap[key]; ok {
		if stringMap[key].expire.IsZero() {
			return Payload{DataType: string(STRING), Str: stringMap[key].value}
		}
		if stringMap[key].expire.Before(time.Now()) {
			delete(stringMap, key)
			return NilValue
		}

		return Payload{DataType: string(STRING), Str: stringMap[key].value}
	}
	return NilValue
}