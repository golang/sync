package syncmultimap

import (
	"testing"
)

func TestMultiMap(t *testing.T) {
	mmap := NewMultiMap()

	// Test Add
	val1, val2, val3 := 1, 2, 3
	mmap.Add("testKey1", val1)
	mmap.Add("testKey1", val2)
	mmap.Add("testKey2", val3)

	// Test RangeKey
	testKey1Sum := 0
	mmap.RangeKey("testKey1", func(key string, v interface{}) bool {
		if key != "testKey1" {
			t.Errorf("Expected key to be 'testKey1', got %s", key)
		}
		testKey1Sum += (v).(int)
		return true
	})

	if testKey1Sum != 3 {
		t.Errorf("Expected sum of values for 'testKey1' to be 3, got %d", testKey1Sum)
	}

	// Test RangeAll
	totalSum := 0
	mmap.RangeAll(func(_ string, v interface{}) bool {
		totalSum += (v).(int)
		return true
	})

	if totalSum != 6 {
		t.Errorf("Expected sum of all values to be 6, got %d", totalSum)
	}

	// Test Remove
	mmap.Remove("testKey1", val2)
	testKey1SumAfterRemove := 0
	mmap.RangeKey("testKey1", func(_ string, v interface{}) bool {
		testKey1SumAfterRemove += (v).(int)
		return true
	})

	if testKey1SumAfterRemove != 1 {
		t.Errorf("Expected sum of values for 'testKey1' after removal to be 1, got %d", testKey1SumAfterRemove)
	}

	// Test RemoveAll
	mmap.RemoveAll("testKey1")
	testKey1SumAfterRemoveAll := 0
	mmap.RangeKey("testKey1", func(_ string, v interface{}) bool {
		testKey1SumAfterRemoveAll += (v).(int)
		return true
	})

	if testKey1SumAfterRemoveAll != 0 {
		t.Errorf("Expected sum of values for 'testKey1' after removeAll to be 0, got %d", testKey1SumAfterRemoveAll)
	}

	// Test ContainsKey
	if !mmap.ContainsKey("testKey2") {
		t.Errorf("Expected map to contain key 'testKey2'")
	}
	if mmap.ContainsKey("testKey1") {
		t.Errorf("Expected map to not contain key 'testKey1'")
	}
}
