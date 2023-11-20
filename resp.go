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

const (
	STRING     = '+'
	ERROR      = '-'
	INTEGER    = ':'
	BULKSTRING = '$'
	ARRAY      = '*'
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

	switch firstByte {
	case ARRAY:
		return r.readArray()
	case BULKSTRING:
		return r.readBulkString()
	default:
		err := fmt.Errorf("Unexpected first byte of payload")
		return Payload{}, err
	}
}

// Expected format 2\r\n<payload>\r\n<payload>\r\n
func (r *RespReader) readArray() (Payload, error) {

	p := Payload{}
	b, err := r.reader.ReadBytes('\n')
	if err != nil {
		return Payload{}, errors.New("wrong payload format. unable to parse size")
	}
	size, _ := strconv.ParseInt(string(b), 10, 64)

	p.array = make([]Payload, 0)
	for i := 0; i < int(size); i++ {
		payload, err := r.readBulkString()
		if err != nil {
			return p, err
		}

		p.array = append(p.array, payload)
	}
	return p, nil
}

// Expected format $\r\n<payload>\r\n
func (r *RespReader) readBulkString() (Payload, error) {

	firstByte, err := r.reader.ReadByte()
	if err != nil {
		err := fmt.Errorf("failed to parse first byte due to %q", err)
		return Payload{}, err
	}

	if firstByte != BULKSTRING {
		err := fmt.Errorf("first byte is not the one expected for bulk")
		return Payload{}, err
	}

	b, err := r.reader.ReadBytes('\n')
	if err != nil {
		return Payload{}, errors.New("wrong payload format. unable to parse size")
	}
	size, _ := strconv.ParseInt(string(b), 10, 64)

	p := Payload{}

	p.bulk, err = r.reader.ReadString('\n')
	if int64(len(p.bulk)) != size {
		return Payload{}, errors.New("wrong payload format. bulk string size is not the same as the size in the payload")
	}

	return p, nil
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
