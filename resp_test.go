package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerialize(t *testing.T) {

	respString := "$5\r\nhello\r\n"
	res, err := deserialize(respString)

	require.NoError(t, err)
	require.Equal(t, res.str, "hello")

	// respString = "*-1\r\n"
	// res, err = deserialize(respString)

	// require.NoError(t, err)
	// require.Equal(t, res.isNull, true)
}
