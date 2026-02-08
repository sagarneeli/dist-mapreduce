package worker

import (
	"testing"
)

func TestMapFunc(t *testing.T) {
	filename := "test.txt"
	contents := "hello world hello"
	expected := []KeyValue{
		{Key: "hello", Value: "1"},
		{Key: "world", Value: "1"},
		{Key: "hello", Value: "1"},
	}

	result := MapFunc(filename, contents)

	if len(result) != len(expected) {
		t.Fatalf("Expected %d keys, got %d", len(expected), len(result))
	}

	for i, kv := range result {
		if kv != expected[i] {
			t.Errorf("Expected %v at index %d, got %v", expected[i], i, kv)
		}
	}
}

func TestMapFunc_SpecialCharacters(t *testing.T) {
	filename := "test.txt"
	contents := "hello, world! hello."
	expected := []KeyValue{
		{Key: "hello", Value: "1"},
		{Key: "world", Value: "1"},
		{Key: "hello", Value: "1"},
	}

	result := MapFunc(filename, contents)

	if len(result) != len(expected) {
		t.Fatalf("Expected %d keys, got %d", len(expected), len(result))
	}
}

func TestReduceFunc(t *testing.T) {
	key := "hello"
	values := []string{"1", "1", "1"}
	expected := "3"

	result := ReduceFunc(key, values)

	if result != expected {
		t.Errorf("Expected count %s for key %s, got %s", expected, key, result)
	}
}

func TestIHash(t *testing.T) {
	// Property test: same key should hash to same value
	k1 := "hello"
	k2 := "hello"
	if ihash(k1) != ihash(k2) {
		t.Error("Hash function is not deterministic")
	}

	// Different keys (likely) different hash
	k3 := "world"
	if ihash(k1) == ihash(k3) {
		t.Log("Hash collision observed (unlikely but possible)")
	}
}
