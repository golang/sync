package syncmultimap

import (
	sync "sync"
)

type MultiMapSet struct {
	key    string
	values []interface{}
	mapRef *MultiMap
	sync.RWMutex
}

func NewMultiMapSet(mapRef *MultiMap, key string) *MultiMapSet {
	values := make([]interface{}, 0)
	return &MultiMapSet{
		key:    key,
		values: values,
		mapRef: mapRef,
	}
}

func (s *MultiMapSet) Add(value interface{}) {
	s.Lock()
	defer s.Unlock()
	s.values = append(s.values, value)
}

func (s *MultiMapSet) Remove(value interface{}) {
	s.Lock()
	defer s.Unlock()

	// Remove the pointer from the MultiMapSet
	for i, v := range s.values {
		if v == value {
			s.values = append(s.values[:i], s.values[i+1:]...)
			return
		}
	}

	// If the set is now empty, remove it from the map
	if len(s.values) == 0 {
		s.mapRef.RemoveAll(s.key)
	}

}

type MultiMapRangeFn func(string, interface{}) bool

func (s *MultiMapSet) Range(fn MultiMapRangeFn) {
	s.RLock()
	defer s.RUnlock()
	for _, value := range s.values {
		if !fn(s.key, value) {
			return
		}
	}
}

type MultiMap struct {
	values *sync.Map
}

func NewMultiMap() *MultiMap {
	values := sync.Map{}
	return &MultiMap{
		values: &values,
	}
}

func (m *MultiMap) Add(key string, value interface{}) {
	// Ensure a set exists
	multiMapSet := NewMultiMapSet(m, key)
	multiMapSetObj, _ := m.values.LoadOrStore(key, multiMapSet)
	multiMapSet = multiMapSetObj.(*MultiMapSet)

	// Add the value to the set
	multiMapSet.Add(value)

	// Ensure the set is still stored in the map
	// This avoids a race condition whereby the set is removed
	// concurrently.
	m.values.Store(key, multiMapSet)
}

func (m *MultiMap) Remove(key string, value interface{}) {
	multiMapSetObj, ok := m.values.Load(key)
	if !ok {
		return
	}
	multiMapSet := multiMapSetObj.(*MultiMapSet)
	multiMapSet.Remove(value)
}

func (m *MultiMap) RemoveAll(key string) {
	m.values.Delete(key)
}

func (m *MultiMap) RangeKey(key string, fn MultiMapRangeFn) {
	multiMapSetObj, ok := m.values.Load(key)
	if !ok {
		return
	}
	multiMapSet := multiMapSetObj.(*MultiMapSet)
	multiMapSet.Range(fn)
}

func (m *MultiMap) RangeAll(fn MultiMapRangeFn) {
	m.values.Range(func(_, value interface{}) bool {
		multiMapSet := value.(*MultiMapSet)
		multiMapSet.Range(fn)
		return true
	})
}

func (m *MultiMap) ContainsKey(key string) bool {
	_, ok := m.values.Load(key)
	return ok
}
