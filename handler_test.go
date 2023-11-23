package main

import (
	"testing"
	"time"
)

func TestPing(t *testing.T) {
	// Test with no parameters
	response := ping([]Payload{})
	if response.DataType != string(STRING) || response.Str != "PONG" {
		t.Errorf("Expected PONG, got %s", response.Str)
	}

	// Test with a parameter
	testStr := "hello"
	response = ping([]Payload{{DataType: string(BULKSTRING), Bulk: testStr}})
	if response.DataType != string(STRING) || response.Str != testStr {
		t.Errorf("Expected %s, got %s", testStr, response.Str)
	}
}

func TestEcho(t *testing.T) {
	// Test with valid parameter
	testStr := "hello"
	response := echo([]Payload{{DataType: string(BULKSTRING), Bulk: testStr}})
	if response.DataType != string(STRING) || response.Str != testStr {
		t.Errorf("Expected %s, got %s", testStr, response.Str)
	}

	// Test with no parameters
	response = echo([]Payload{})
	if response.DataType != string(ERROR) {
		t.Errorf("Expected error, got %s", response.DataType)
	}

	// Test with multiple parameters
	response = echo([]Payload{{Bulk: "one"}, {Bulk: "two"}})
	if response.DataType != string(ERROR) {
		t.Errorf("Expected error, got %s", response.DataType)
	}
}

func TestCommand(t *testing.T) {
	response := command([]Payload{})
	if response.DataType != string(ARRAY) || len(response.Array) == 0 {
		t.Errorf("Expected non-empty array, got %v", response.Array)
	}
}

func TestSet(t *testing.T) {
	// Test setting a key-value pair
	response := set([]Payload{{Bulk: "key1"}, {Bulk: "value1"}})
	if response.DataType != string(STRING) || response.Str != "OK" {
		t.Errorf("Expected OK, got %s", response.Str)
	}

	// Test setting a key-value pair with expiration
	response = set([]Payload{{Bulk: "key2"}, {Bulk: "value2"}, {Bulk: "EX"}, {Bulk: "10"}})
	if response.DataType != string(STRING) || response.Str != "OK" {
		t.Errorf("Expected OK, got %s", response.Str)
	}

	// Test with invalid expiration time format
	response = set([]Payload{{Bulk: "key3"}, {Bulk: "value3"}, {Bulk: "EX"}, {Bulk: "invalid"}})
	if response.DataType != string(STRING) || response.Str != "OK" {
		t.Errorf("Expected OK, got %s", response.Str)
	}
}

func TestGet(t *testing.T) {
	// Setting up test data
	stringMap["key1"] = stringValue{"value1", time.Time{}}
	stringMap["key2"] = stringValue{"value2", time.Now().Add(time.Second)}

	// Test getting an existing key
	response := get([]Payload{{Bulk: "key1"}})
	if response.DataType != string(STRING) || response.Str != "value1" {
		t.Errorf("Expected value1, got %s", response.Str)
	}

	// Test getting a non-existing key
	response = get([]Payload{{Bulk: "nonexisting"}})
	if response.Bulk != NilValue.Bulk {
		t.Errorf("Expected -1, got %s", response.Bulk)
	}

	// Test getting an expired key
	time.Sleep(time.Second * 2)
	response = get([]Payload{{Bulk: "key2"}})
	if response.Bulk != NilValue.Bulk {
		t.Errorf("Expected -1, got %s", response.Bulk)
	}
}

func TestExist(t *testing.T) {
	// Setting up test data
	stringMap["key1"] = stringValue{"value1", time.Time{}}
	stringMap["key2"] = stringValue{"value2", time.Now().Add(time.Second)}

	// Test with existing keys
	response := exist([]Payload{{Bulk: "key1"}, {Bulk: "key2"}})
	if response.DataType != string(INTEGER) || response.Num != 2 {
		t.Errorf("Expected 2, got %d", response.Num)
	}

	// Test with non-existing keys
}
