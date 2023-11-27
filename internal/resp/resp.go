package resp

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
	DataType string
	Str      string
	Num      int
	Bulk     string
	Array    []Payload
}

var NilValue = Payload{DataType: string(ARRAY), Bulk: "-1"}

type RespReader struct {
	reader bufio.Reader
}

type RespWriter struct {
	writer bufio.Writer
}

func NewRespReader(rd io.Reader) *RespReader {
	return &RespReader{reader: *bufio.NewReader(rd)}
}

func NewRespWriter(wr io.Writer) *RespWriter {
	return &RespWriter{writer: *bufio.NewWriter(wr)}
}

// Parse payload that follows RESP protocol into payload struct
// Array of Bulk strings is expected
func (r *RespReader) Read() (Payload, error) {

	firstByte, err := r.reader.ReadByte()
	if err != nil {
		if err != io.EOF {
			err = fmt.Errorf("failed to parse first byte due to %q", err)
		}
		return Payload{}, err
	}
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
	b, _, err := r.reader.ReadLine()
	if err != nil {
		return Payload{}, errors.New("wrong payload format. unable to parse size")
	}
	size, _ := strconv.ParseInt(string(b), 10, 64)
	// Null value is represented as "*-1\r\n"
	if size == -1 {
		return p, nil
	}

	p.Array = make([]Payload, 0)
	for i := 0; i < int(size); i++ {
		payload, err := r.Read()
		if err != nil {
			return p, err
		}
		p.Array = append(p.Array, payload)
	}
	p.DataType = string(ARRAY)
	return p, nil
}

// Expected format $\r\n<payload>\r\n
func (r *RespReader) readBulkString() (Payload, error) {

	p := Payload{}
	b, _, err := r.reader.ReadLine()
	if err != nil {
		return p, errors.New("wrong payload format. unable to parse size")
	}
	size, _ := strconv.ParseInt(string(b), 10, 64)
	// Null value is represented as "$-1\r\n"
	if size == -1 {
		return p, nil
	}
	b, _, err = r.reader.ReadLine()
	if err != nil || int64(len(b)) != size {
		return Payload{}, errors.New("wrong payload format. bulk string size is not the same as the size in the payload")
	}

	p.Bulk = string(b)
	p.DataType = string(BULKSTRING)
	return p, nil
}

func (p *Payload) WriteString() []byte {
	bytes := make([]byte, 0)
	bytes = append(bytes, STRING)
	bytes = append(bytes, p.Str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (p *Payload) WriteErrors() []byte {
	bytes := make([]byte, 0)
	bytes = append(bytes, ERROR)
	bytes = append(bytes, []byte("Error ")...)
	bytes = append(bytes, p.Str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (p *Payload) WriteIntegers() []byte {
	bytes := make([]byte, 0)
	bytes = append(bytes, INTEGER)
	bytes = append(bytes, []byte(strconv.Itoa(p.Num))...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}
func (p *Payload) WriteBulkString() []byte {
	bytes := make([]byte, 0)
	bytes = append(bytes, BULKSTRING)
	bytes = append(bytes, []byte(strconv.Itoa(len(p.Bulk)))...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, []byte(p.Bulk)...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (p *Payload) WriteArray() []byte {
	bytes := make([]byte, 0)
	bytes = append(bytes, ARRAY)
	bytes = append(bytes, []byte(strconv.Itoa(len(p.Array)))...)
	bytes = append(bytes, '\r', '\n')
	for i := 0; i < len(p.Array); i++ {
		bytes = append(bytes, p.Array[i].Write()...)
	}

	return bytes
}

func (p *Payload) Write() []byte {
	var bytes []byte
	switch p.DataType {
	case string(STRING):
		bytes = p.WriteString()
	case string(ERROR):
		bytes = p.WriteErrors()
	case string(INTEGER):
		bytes = p.WriteIntegers()
	case string(BULKSTRING):
		bytes = p.WriteBulkString()
	case string(ARRAY):
		bytes = p.WriteArray()
	default:
		bytes = []byte("*-1\r\n")
	}

	return bytes
}

func (w *RespWriter) Write(p *Payload) error {
	_, err := w.writer.Write(p.Write())
	if err != nil {
		return err
	}
	err = w.writer.Flush()
	return err
}

func ParseRequest(cmd *Payload) (string, []Payload) {
	// first bulk string is the command
	request := strings.ToUpper(cmd.Array[0].Bulk)
	params := cmd.Array[1:]

	return request, params
}
