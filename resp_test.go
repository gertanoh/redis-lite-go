package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerialize(t *testing.T) {

	t.Run("Bulk String", func(t *testing.T) {
		bulkString := "$5\r\nhello\r\n"
		respReader := NewRespReader(strings.NewReader(bulkString))

		res, err := respReader.Read()
		require.NoError(t, err)
		require.Equal(t, "hello", res.Bulk)
	})

}
