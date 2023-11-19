package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Module to serialize and deserialize RESP protocol messages
//

const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

type Payload struct {
	dataType string
	str      string
	num      int
	bulk     string
	array    []Payload
}

type RespReader struct {
	reader bufio.Reader
}

func NewRespReader(rd io.Reader) *RespReader {
	return &RespReader{reader: *bufio.NewReader(rd)}
}

// Read does the deserialization process
func (r *RespReader) Read() (Payload, error) {

	firstByte, err := r.reader.ReadByte()
	if err != nil {
		err := fmt.Errorf("failed to parse first byte due to %q", err)
		return Payload{}, err
	}

	if firstByte != ARRAY {
		return Payload{}, errors.New("wrong payload format. Expecting an array of bulk string")
	}

	b, _ := r.reader.ReadByte()
	size, _ := strconv.ParseInt(string(b), 10, 64)

	// consume /r/n
	r.reader.ReadByte()
	r.reader.ReadByte()

	// Parse each line using scanner
	scanner := bufio.NewScanner(&r.reader)
	var count int64
	for scanner.Scan() {
		count += 1
		bulkString := scanner.Text()
	}

	if count != size {
		return Payload{}, errors.New("wrong format. Received size is not the same")
	}
}

func (r *RespReader) ReadBulkString() (Payload, error) {
	
}
func retrieveUsefulData(s string) string {
	index := strings.Index(s, "\r\n")
	if index == -1 {
		return string("")
	}
	return s[:index]
}

// Parse payload that follows RESP protocol into payload struct
// Array of Bulk strings is expected
func deserialize(msg string) (*Payload, error) {

	value := &Payload{}

	reader := bufio.NewReader(strings.NewReader(msg))

	b, err := reader.ReadByte()
	if err != nil {
		return value, errors.New("msg format not valid, not able to read first byte")
	}

	if b != '$' {
		return value, errors.New("msg format not valid, not able to read first byte")
	}

	// read size
	b, _ = reader.ReadByte()

	size, _ := strconv.ParseInt(string(b), 10, 64)

	// consume line carriage
	reader.ReadByte()
	reader.ReadByte()

	data := make([]byte, size)

	_, err = reader.Read(data)
	if err != nil {
		return nil, errors.New("msg format not valid, not able to read bulk string")
	}

	fmt.Println(data)

	value.dataType = STRING
	value.str = string(data)

	fmt.Println(value.str)

	return value, nil
}
