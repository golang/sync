// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package singleflight

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testConcurrentHelper(t *testing.T, inGoroutine func(routineIndex, goroutineCount int)) {
	var wg, wgGoroutines sync.WaitGroup
	const callers = 4
	//ref := make([]RefCounter, callers)
	wgGoroutines.Add(callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			wgGoroutines.Done()
			wgGoroutines.Wait() // ensure that all goroutines started and reached this point

			inGoroutine(i, callers)
		}(i)
	}
	wg.Wait()

}

func TestUse(t *testing.T) {
	var g Group
	var newCount, handleCount, disposeCount int64

	testConcurrentHelper(
		t,
		func(index, goroutineCount int) {
			g.Use(
				"key",
				// 'new' is a slow function that creates a temp resource
				func() (interface{}, error) {
					time.Sleep(200 * time.Millisecond) // let more goroutines enter Do
					atomic.AddInt64(&newCount, 1)
					return "bar", nil
				},
				// 'fn' to be called by each goroutine
				func(s interface{}, e error) error {
					// handle s
					if newCount != 1 {
						t.Errorf("goroutine %v: newCount(%v) expected to be set prior to this function getting called", index, newCount)
					}
					atomic.AddInt64(&handleCount, 1)
					if disposeCount > 0 {
						t.Errorf("goroutine %v: disposeCount(%v) should not be incremented until all fn are completed", index, disposeCount)
					}
					return e
				},
				// 'dispose' - to be called once at the end
				func(s interface{}) {
					// cleaning up "bar"
					atomic.AddInt64(&disposeCount, 1)
					if handleCount != int64(goroutineCount) {
						t.Errorf("dispose is expected to be called when all %v fn been completed, but %v have been completed instead", goroutineCount, handleCount)
					}
				},
			)
		},
	)

	if newCount != 1 {
		t.Errorf("new expected to be called exactly once, was called %v", newCount)
	}
	if disposeCount != 1 {
		t.Errorf("dispose expected to be called exactly once, was called %v", disposeCount)
	}
}

func TestUseWithResource(t *testing.T) {
	// use this "global" var for checkes after that testConcurrentHelper call
	var tempFileName string

	var g Group
	testConcurrentHelper(
		t,
		func(_, _ int) {
			g.Use(
				"key",
				// 'new' is a slow function that creates a temp resource
				func() (interface{}, error) {
					time.Sleep(200 * time.Millisecond) // let more goroutines enter Do
					f, e := ioutil.TempFile("", "pat")
					if e != nil {
						return nil, e
					}
					defer f.Close()
					tempFileName = f.Name()

					// fill temp file with sequence of n.Write(...) calls

					return f.Name(), e
				},
				// 'fn' to be called by each goroutine
				func(s interface{}, e error) error {
					// handle s
					if e != nil {
						// send alternative payload
					}
					if e == nil {
						/*tempFileName*/ _ = s.(string)
						// send Content of tempFileName to HTTPWriter
					}
					return e
				},
				// 'dispose' - to be called once at the end
				func(s interface{}) {
					// cleaning up "bar"
					os.Remove(s.(string))
				},
			)
		},
	)
	if _, e := os.Stat(tempFileName); !os.IsNotExist(e) {
		t.Errorf("test has created a temp file '%v', but failed to cleaned it", tempFileName)
	}
}

func TestDo(t *testing.T) {
	var g Group
	v, err, _ := g.Do("key", func() (interface{}, error) {
		return "bar", nil
	})
	if got, want := fmt.Sprintf("%v (%T)", v, v), "bar (string)"; got != want {
		t.Errorf("Do = %v; want %v", got, want)
	}
	if err != nil {
		t.Errorf("Do error = %v", err)
	}
}

func TestDoErr(t *testing.T) {
	var g Group
	someErr := errors.New("Some error")
	v, err, _ := g.Do("key", func() (interface{}, error) {
		return nil, someErr
	})
	if err != someErr {
		t.Errorf("Do error = %v; want someErr %v", err, someErr)
	}
	if v != nil {
		t.Errorf("unexpected non-nil value %#v", v)
	}
}

func TestDoDupSuppress(t *testing.T) {
	var g Group
	var wg1, wg2 sync.WaitGroup
	c := make(chan string, 1)
	var calls int32
	fn := func() (interface{}, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			// First invocation.
			wg1.Done()
		}
		v := <-c
		c <- v // pump; make available for any future calls

		time.Sleep(10 * time.Millisecond) // let more goroutines enter Do

		return v, nil
	}

	const n = 10
	wg1.Add(1)
	for i := 0; i < n; i++ {
		wg1.Add(1)
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			wg1.Done()
			v, err, _ := g.Do("key", fn)
			if err != nil {
				t.Errorf("Do error: %v", err)
				return
			}
			if s, _ := v.(string); s != "bar" {
				t.Errorf("Do = %T %v; want %q", v, v, "bar")
			}
		}()
	}
	wg1.Wait()
	// At least one goroutine is in fn now and all of them have at
	// least reached the line before the Do.
	c <- "bar"
	wg2.Wait()
	if got := atomic.LoadInt32(&calls); got <= 0 || got >= n {
		t.Errorf("number of calls = %d; want over 0 and less than %d", got, n)
	}
}

// Test that singleflight behaves correctly after Forget called.
// See https://github.com/golang/go/issues/31420
func TestForget(t *testing.T) {
	var g Group

	var firstStarted, firstFinished sync.WaitGroup

	firstStarted.Add(1)
	firstFinished.Add(1)

	firstCh := make(chan struct{})
	go func() {
		g.Do("key", func() (i interface{}, e error) {
			firstStarted.Done()
			<-firstCh
			firstFinished.Done()
			return
		})
	}()

	firstStarted.Wait()
	g.Forget("key") // from this point no two function using same key should be executed concurrently

	var secondStarted int32
	var secondFinished int32
	var thirdStarted int32

	secondCh := make(chan struct{})
	secondRunning := make(chan struct{})
	go func() {
		g.Do("key", func() (i interface{}, e error) {
			defer func() {
			}()
			atomic.AddInt32(&secondStarted, 1)
			// Notify that we started
			secondCh <- struct{}{}
			// Wait other get above signal
			<-secondRunning
			<-secondCh
			atomic.AddInt32(&secondFinished, 1)
			return 2, nil
		})
	}()

	close(firstCh)
	firstFinished.Wait() // wait for first execution (which should not affect execution after Forget)

	<-secondCh
	// Notify second that we got the signal that it started
	secondRunning <- struct{}{}
	if atomic.LoadInt32(&secondStarted) != 1 {
		t.Fatal("Second execution should be executed due to usage of forget")
	}

	if atomic.LoadInt32(&secondFinished) == 1 {
		t.Fatal("Second execution should be still active")
	}

	close(secondCh)
	result, _, _ := g.Do("key", func() (i interface{}, e error) {
		atomic.AddInt32(&thirdStarted, 1)
		return 3, nil
	})

	if atomic.LoadInt32(&thirdStarted) != 0 {
		t.Error("Third call should not be started because was started during second execution")
	}
	if result != 2 {
		t.Errorf("We should receive result produced by second call, expected: 2, got %d", result)
	}
}
