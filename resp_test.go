package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRespReader(t *testing.T) {

	t.Run("Bulk String", func(t *testing.T) {
		bulkString := "$5\r\nhello\r\n"
		respReader := NewRespReader(strings.NewReader(bulkString))

		res, err := respReader.Read()
		require.NoError(t, err)
		require.Equal(t, "hello", res.Bulk)
	})

	t.Run("Null Bulk String", func(t *testing.T) {
		bulkString := "$0\r\n\r\n"
		respReader := NewRespReader(strings.NewReader(bulkString))

		res, err := respReader.Read()
		require.NoError(t, err)
		require.Equal(t, "", res.Bulk)
	})
	t.Run("Null value as bulk String", func(t *testing.T) {
		bulkString := "$-1\r\n"
		respReader := NewRespReader(strings.NewReader(bulkString))

		res, err := respReader.Read()
		require.NoError(t, err)
		require.Equal(t, "", res.Bulk)
	})

	t.Run("Array of bulk string", func(t *testing.T) {
		bulkString := "*2\r\n$4\r\necho\r\n$11\r\nhello-world\r\n"
		respReader := NewRespReader(strings.NewReader(bulkString))

		res, err := respReader.Read()
		require.NoError(t, err)

		require.Equal(t, len(res.Array), 2)
		require.Equal(t, "echo", res.Array[0].Bulk)
		require.Equal(t, "hello-world", res.Array[1].Bulk)
	})

	t.Run("Invalid Bulk String", func(t *testing.T) {
		bulkString := "*2\r\n$17\r\necho\r\n$11\r\nhello-world\r\n"
		respReader := NewRespReader(strings.NewReader(bulkString))

		_, err := respReader.Read()
		require.Error(t, err)
	})
}

// Helper function to create a writer for testing
func createTestRespWriter() (*RespWriter, *bytes.Buffer) {
	var buf bytes.Buffer
	writer := NewRespWriter(&buf)
	return writer, &buf
}

func TestRespWriter(t *testing.T) {

	t.Run("Write string", func(t *testing.T) {
		writer, buf := createTestRespWriter()
		payload := Payload{DataType: string(STRING), Str: "OK"}

		if err := writer.Write(&payload); err != nil {
			t.Fatalf("Write returned an error: %v", err)
		}

		expected := "+OK\r\n"
		require.Equal(t, expected, buf.String())
	})
	t.Run("Write error", func(t *testing.T) {
		writer, buf := createTestRespWriter()
		payload := Payload{DataType: string(ERROR), Str: "WRONGTYPE Operation against a key holding the wrong kind of value"}

		if err := writer.Write(&payload); err != nil {
			t.Fatalf("Write returned an error: %v", err)
		}

		expected := "-Error WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
		require.Equal(t, expected, buf.String())
	})

	t.Run("Write integer", func(t *testing.T) {
		writer, buf := createTestRespWriter()
		payload := Payload{DataType: string(INTEGER), Num: 0}

		if err := writer.Write(&payload); err != nil {
			t.Fatalf("Write returned an error: %v", err)
		}

		expected := ":" + strconv.Itoa(0) + "\r\n"
		require.Equal(t, expected, buf.String())
	})

	t.Run("Write bulk string", func(t *testing.T) {
		writer, buf := createTestRespWriter()
		payload := Payload{DataType: string(BULKSTRING), Bulk: "bulk data"}

		if err := writer.Write(&payload); err != nil {
			t.Fatalf("Write returned an error: %v", err)
		}

		expected := "$9\r\nbulk data\r\n"
		require.Equal(t, expected, buf.String())
	})

	t.Run("Write Array", func(t *testing.T) {
		writer, buf := createTestRespWriter()
		payload := Payload{
			DataType: string(ARRAY),
			Array: []Payload{
				{DataType: string(STRING), Str: "elem1"},
				{DataType: string(INTEGER), Num: 2},
			},
		}

		if err := writer.Write(&payload); err != nil {
			t.Fatalf("Write returned an error: %v", err)
		}

		expected := "*2\r\n+elem1\r\n:2\r\n"
		require.Equal(t, expected, buf.String())
	})

	t.Run("Write Array", func(t *testing.T) {
		writer, buf := createTestRespWriter()
		payload := Payload{
			DataType: string(ARRAY),
			Array: []Payload{
				{DataType: string(BULKSTRING), Bulk: "first"},
				{DataType: string(BULKSTRING), Bulk: "second"},
			},
		}

		if err := writer.Write(&payload); err != nil {
			t.Fatalf("Write returned an error: %v", err)
		}

		// The expected output depends on the RESP format for arrays and bulk strings.
		// It should look something like this:
		// *2\r\n$5\r\nfirst\r\n$6\r\nsecond\r\n
		expected := "*2\r\n$5\r\nfirst\r\n$6\r\nsecond\r\n"
		require.Equal(t, expected, buf.String())
	})
}
