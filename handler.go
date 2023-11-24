package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var handlers = map[string]func([]Payload) Payload{
	"PING":    ping,
	"COMMAND": command,
	"ECHO":    echo,
	"SET":     set,
	"GET":     get,
	"EXISTS":  exist,
	"DEL":     del,
	"INCR":    incr,
}

type stringValue struct {
	value  string
	expire time.Time
}

var stringMap = map[string]stringValue{}
var stringMapLock sync.RWMutex

func updateInMemoryStore(request string, params []Payload) Payload {
	var response Payload
	if _, ok := handlers[request]; ok {
		response = handlers[request](params)
	} else {
		response.DataType = string(ERROR)
		response.Str = "Unknown command"
	}
	return response
}

func parseRequest(cmd *Payload) (string, []Payload) {
	// first bulk string is the command
	request := strings.ToUpper(cmd.Array[0].Bulk)
	params := cmd.Array[1:]

	return request, params
}

func processRequest(cmd *Payload, aof *Aof) Payload {
	p := Payload{}
	if cmd.DataType != string(ARRAY) {
		p.DataType = string(ERROR)
		p.Str = "Expected array of bulk strings"
		return p
	}

	if len(cmd.Array) == 0 {
		p.DataType = string(ERROR)
		p.Str = "Null array command"
		return p
	}

	request, params := parseRequest(cmd)
	if request == "SET" || request == "INCR"{
		aof.Write(cmd)
	}

	response := updateInMemoryStore(request, params)

	return response
}

func HandleConnection(conn net.Conn, aof *Aof) {

	defer conn.Close()
	respReader := NewRespReader(conn)
	writer := NewRespWriter(conn)

	for {
		cmd, err := respReader.Read()
		var response Payload

		if err != nil {
			if err != io.EOF {
				log.Println(err)
				response.DataType = string(ERROR)
				response.Str = "Invalid request format"
			} else {
				return
			}
		} else {
			response = processRequest(&cmd, aof)
		}

		err = writer.Write(&response)
		if err != nil {
			log.Println("writer : ", err)
		}
	}
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
		{DataType: string(BULKSTRING), Bulk: "ECHO"},
		{DataType: string(BULKSTRING), Bulk: "COMMAND"},
		{DataType: string(BULKSTRING), Bulk: "PING"},
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
			if err == nil {
				expire = time.Now().Add(time.Duration(expireInSecs) * time.Second)
			}
		}
	}
	stringMapLock.Lock()
	defer stringMapLock.Unlock()
	stringMap[key] = stringValue{value, expire}
	return Payload{DataType: string(STRING), Str: "OK"}
}

func get(p []Payload) Payload {
	key := p[0].Bulk
	stringMapLock.RLock()
	defer stringMapLock.RUnlock()
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

func exist(p []Payload) Payload {
	stringMapLock.RLock()
	defer stringMapLock.RUnlock()

	var count int

	for i := 0; i < len(p); i++ {
		key := p[i].Bulk
		if _, ok := stringMap[key]; ok {
			if stringMap[key].expire.IsZero() {
				count++
			} else if stringMap[key].expire.After(time.Now()) {
				count++
			}
		}
	}
	return Payload{DataType: string(INTEGER), Num: count}
}

func del(p []Payload) Payload {
	stringMapLock.Lock()
	defer stringMapLock.Unlock()

	var count int

	for i := 0; i < len(p); i++ {
		key := p[i].Bulk
		if _, ok := stringMap[key]; ok {
			delete(stringMap, key)
			count++
		}
	}
	return Payload{DataType: string(INTEGER), Num: count}
}

func incr(p []Payload) Payload {
	stringMapLock.Lock()
	defer stringMapLock.Unlock()

	var count int
	var strValue string

	key := p[0].Bulk
	if _, ok := stringMap[key]; ok {
		if stringMap[key].expire.IsZero() {
			strValue = stringMap[key].value
		}
		if stringMap[key].expire.Before(time.Now()) {
			delete(stringMap, key)
		} else {
			strValue = stringMap[key].value
		}
	}
	if strValue != "" {
		countOn64, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return Payload{DataType: string(ERROR), Str: "Key value is not integer"}
		}
		count = int(countOn64)
	}
	count++
	countStrValue := strconv.Itoa(count)
	stringMap[key] = stringValue{value: countStrValue}
	return Payload{DataType: string(INTEGER), Num: count}
}
