package handler

import (
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/ger/redis-lite-go/internal/resp"
)

var handlers = map[string]func([]resp.Payload) resp.Payload{
	"PING":    ping,
	"COMMAND": command,
	"ECHO":    echo,
	"SET":     set,
	"GET":     get,
	"EXISTS":  exist,
	"DEL":     del,
	"INCR":    incr,
	"HSET":    hset,
	"HGET":    hget,
}

type stringValue struct {
	value  string
	expire time.Time
}

var stringMap = map[string]stringValue{}
var hashMap = map[string]map[string]stringValue{}

var stringMapLock sync.RWMutex
var hashMapLock sync.RWMutex

func updateInMemoryStore(request string, params []resp.Payload) resp.Payload {
	var response resp.Payload
	if _, ok := handlers[request]; ok {
		response = handlers[request](params)
	} else {
		response.DataType = string(resp.ERROR)
		response.Str = "Unknown command"
	}
	return response
}

func processRequest(cmd *resp.Payload, aof *Aof) resp.Payload {
	p := resp.Payload{}
	if cmd.DataType != string(resp.ARRAY) {
		p.DataType = string(resp.ERROR)
		p.Str = "Expected array of bulk strings"
		return p
	}

	if len(cmd.Array) == 0 {
		p.DataType = string(resp.ERROR)
		p.Str = "Null array command"
		return p
	}

	request, params := resp.ParseRequest(cmd)
	if request == "SET" || request == "INCR" || request == "HSET" {
		aof.Write(cmd)
	}

	response := updateInMemoryStore(request, params)

	return response
}

func HandleConnection(conn net.Conn, aof *Aof) {

	defer conn.Close()
	respReader := resp.NewRespReader(conn)
	writer := resp.NewRespWriter(conn)

	for {
		cmd, err := respReader.Read()
		var response resp.Payload

		if err != nil {
			if err != io.EOF {
				log.Println(err)
				response.DataType = string(resp.ERROR)
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

func ping(p []resp.Payload) resp.Payload {
	if len(p) == 0 {
		return resp.Payload{DataType: string(resp.STRING), Str: "PONG"}
	}
	return resp.Payload{DataType: string(resp.BULKSTRING), Bulk: p[0].Bulk}
}

func echo(p []resp.Payload) resp.Payload {
	if len(p) != 1 {
		return resp.Payload{DataType: string(resp.ERROR), Str: "Missing arguments for command"}
	}
	return resp.Payload{DataType: string(resp.BULKSTRING), Bulk: p[0].Bulk}
}

func command(p []resp.Payload) resp.Payload {
	return resp.Payload{DataType: string(resp.ARRAY), Array: []resp.Payload{
		{DataType: string(resp.BULKSTRING), Bulk: "ECHO"},
		{DataType: string(resp.BULKSTRING), Bulk: "COMMAND"},
		{DataType: string(resp.BULKSTRING), Bulk: "PING"},
		{DataType: string(resp.BULKSTRING), Bulk: "HGET"},
		{DataType: string(resp.BULKSTRING), Bulk: "HSET"},
		{DataType: string(resp.BULKSTRING), Bulk: "HGETALL"},
	}}
}

func set(p []resp.Payload) resp.Payload {

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
	return resp.Payload{DataType: string(resp.STRING), Str: "OK"}
}

func get(p []resp.Payload) resp.Payload {
	key := p[0].Bulk
	stringMapLock.RLock()
	defer stringMapLock.RUnlock()
	if _, ok := stringMap[key]; ok {
		if stringMap[key].expire.IsZero() {
			return resp.Payload{DataType: string(resp.STRING), Str: stringMap[key].value}
		}
		if stringMap[key].expire.Before(time.Now()) {
			delete(stringMap, key)
			return resp.NilValue
		}

		return resp.Payload{DataType: string(resp.STRING), Str: stringMap[key].value}
	}
	return resp.NilValue
}

func exist(p []resp.Payload) resp.Payload {
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
	return resp.Payload{DataType: string(resp.INTEGER), Num: count}
}

func del(p []resp.Payload) resp.Payload {
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
	return resp.Payload{DataType: string(resp.INTEGER), Num: count}
}

func incr(p []resp.Payload) resp.Payload {
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
			return resp.Payload{DataType: string(resp.ERROR), Str: "Key value is not integer"}
		}
		count = int(countOn64)
	}
	count++
	countStrValue := strconv.Itoa(count)
	stringMap[key] = stringValue{value: countStrValue}
	return resp.Payload{DataType: string(resp.INTEGER), Num: count}
}

func hset(p []resp.Payload) resp.Payload {
	var count int
	hashKey := p[0].Bulk

	hashMapLock.Lock()
	defer hashMapLock.Unlock()
	for i := 1; i < len(p); i += 2 {
		key := p[i].Bulk
		var expire time.Time
		value := stringValue{p[i+1].Bulk, expire}
		if _, ok := hashMap[hashKey]; !ok {
			hashMap[hashKey] = map[string]stringValue{}
		}
		hashMap[hashKey][key] = value
		count++
	}
	return resp.Payload{DataType: string(resp.INTEGER), Num: count}
}

func hget(p []resp.Payload) resp.Payload {

	if len(p) < 2 {
		return resp.Payload{DataType: string(resp.ERROR), Str: "Missing arguments for command"}
	}
	hashKey := p[0].Bulk
	mapKey := p[1].Bulk

	hashMapLock.RLock()
	defer hashMapLock.RUnlock()
	if _, ok := hashMap[hashKey]; ok {
		if _, ok := hashMap[hashKey][mapKey]; ok {
			return resp.Payload{DataType: string(resp.BULKSTRING), Bulk: hashMap[hashKey][mapKey].value}
		}
	}
	return resp.NilValue
}
