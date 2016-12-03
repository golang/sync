// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syncmap_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"golang.org/x/sync/syncmap"
)

// mapCall is a quick.Generator for calls on mapInterface.
type mapCall struct {
	key   interface{}
	apply func(mapInterface) (interface{}, bool)
	desc  string
}

type mapResult struct {
	value interface{}
	ok    bool
}

var stringType = reflect.TypeOf("")

func randValue(r *rand.Rand) interface{} {
	k, ok := quick.Value(stringType, r)
	if !ok {
		panic(fmt.Sprintf("quick.Value(%v, _) failed", stringType))
	}
	return k.Interface()
}

func (mapCall) Generate(r *rand.Rand, size int) reflect.Value {
	k := randValue(r)

	var (
		app  func(mapInterface) (interface{}, bool)
		desc string
	)
	switch rand.Intn(4) {
	case 0:
		app = func(m mapInterface) (interface{}, bool) {
			return m.Load(k)
		}
		desc = fmt.Sprintf("Load(%q)", k)

	case 1:
		v := randValue(r)
		app = func(m mapInterface) (interface{}, bool) {
			m.Store(k, v)
			return nil, false
		}
		desc = fmt.Sprintf("Store(%q, %q)", k, v)

	case 2:
		v := randValue(r)
		app = func(m mapInterface) (interface{}, bool) {
			return m.LoadOrStore(k, v)
		}
		desc = fmt.Sprintf("LoadOrStore(%q, %q)", k, v)

	case 3:
		app = func(m mapInterface) (interface{}, bool) {
			m.Delete(k)
			return nil, false
		}
		desc = fmt.Sprintf("Delete(%q)", k)
	}

	return reflect.ValueOf(mapCall{k, app, desc})
}

func applyCalls(m mapInterface, calls []mapCall) (results []mapResult, final map[interface{}]interface{}) {
	for _, c := range calls {
		v, ok := c.apply(m)
		results = append(results, mapResult{v, ok})
	}

	final = make(map[interface{}]interface{})
	m.Range(func(k, v interface{}) bool {
		final[k] = v
		return true
	})

	return results, final
}

func applyMap(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	return applyCalls(new(syncmap.Map), calls)
}

func applyRWMutexMap(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	return applyCalls(new(RWMutexMap), calls)
}

func applyDeepCopyMap(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	return applyCalls(new(DeepCopyMap), calls)
}

func TestMapMatchesRWMutex(t *testing.T) {
	if err := quick.CheckEqual(applyMap, applyRWMutexMap, nil); err != nil {
		t.Error(err)
	}
}

func TestMapMatchesDeepCopy(t *testing.T) {
	if err := quick.CheckEqual(applyMap, applyRWMutexMap, nil); err != nil {
		t.Error(err)
	}
}
