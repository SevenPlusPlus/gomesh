package network

import (
	"testing"
	"fmt"
	"time"
	"sync"
)

func TestWorkerPool_Put(t *testing.T) {
	workerPool := newWorkerPoolWithOpts(2,2)
	jodIdx := 0
	mux := sync.Mutex{}
	if err := workerPool.Put(1, func() {
		var myIdx int
		mux.Lock()
		jodIdx = jodIdx+1
		myIdx = jodIdx
		mux.Unlock()
		fmt.Printf("Job-%d begin\n", myIdx)
		time.Sleep(1 * time.Second)
		fmt.Printf("Job-%d end\n", myIdx)
	}); err != nil {
		fmt.Printf("Got err %v\n", err)
	}
	if err := workerPool.Put(2, func() {
		var myIdx int
		mux.Lock()
		jodIdx = jodIdx+1
		myIdx = jodIdx
		mux.Unlock()
		fmt.Printf("Job-%d begin\n", myIdx)
		time.Sleep(2 * time.Second)
		fmt.Printf("Job-%d end\n", myIdx)
	}); err != nil {
		fmt.Printf("Got err %v\n", err)
	}
	if err := workerPool.Put(1, func() {
		var myIdx int
		mux.Lock()
		jodIdx = jodIdx+1
		myIdx = jodIdx
		mux.Unlock()
		fmt.Printf("Job-%d begin\n", myIdx)
		time.Sleep(1 * time.Second)
		fmt.Printf("Job-%d end\n", myIdx)
	}); err != nil {
		fmt.Printf("Got err %v\n", err)
	}
	if err := workerPool.Put(2, func() {
		var myIdx int
		mux.Lock()
		jodIdx = jodIdx+1
		myIdx = jodIdx
		mux.Unlock()
		fmt.Printf("Job-%d begin\n", myIdx)
		time.Sleep(2 * time.Second)
		fmt.Printf("Job-%d end\n", myIdx)
	}); err != nil {
		fmt.Printf("Got err %v\n", err)
	}

	select {
	case <- time.After(2 * time.Second):
		workerPool.Close()
	}
}
