package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
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
	DataType string
	Str      string
	Num      int
	Bulk     string
	Array    []Payload
}

type RespReader struct {
	reader bufio.Reader
}

func NewRespReader(rd io.Reader) *RespReader {
	return &RespReader{reader: *bufio.NewReader(rd)}
}

// Parse payload that follows RESP protocol into payload struct
// Array of Bulk strings is expected
func (r *RespReader) Read() (Payload, error) {

	firstByte, err := r.reader.ReadByte()
	if err != nil {
		err := fmt.Errorf("failed to parse first byte due to %q", err)
		return Payload{}, err
	}
	fmt.Printf("first byte : %q\n", firstByte)
	switch firstByte {
	case ARRAY:
		return r.readArray()
	case BULKSTRING:
		return r.readBulkString()
	default:
		err := fmt.Errorf("unexpected first byte of payload")
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

	p.Array = make([]Payload, 0)
	for i := 0; i < int(size); i++ {
		payload, err := r.readBulkString()
		if err != nil {
			return p, err
		}

		p.Array = append(p.Array, payload)
	}
	return p, nil
}

// Expected format $\r\n<payload>\r\n
func (r *RespReader) readBulkString() (Payload, error) {

	// Assume that
	b, _, err := r.reader.ReadLine()
	if err != nil {
		return Payload{}, errors.New("wrong payload format. unable to parse size")
	}

	size, _ := strconv.ParseInt(string(b), 10, 64)

	p := Payload{}

	b, _, err = r.reader.ReadLine()

	if err != nil || int64(len(b)) != size {
		return Payload{}, errors.New("wrong payload format. bulk string size is not the same as the size in the payload")
	}

	p.Bulk = string(b)
	return p, nil
}
