/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWrappedInstance(t *testing.T) {

	wasmBytes, contractId, logger := prepareContract("./testdata/rust-counter-1.2.0.wasm", t)

	vmPool, err := newVmPool(&contractId, wasmBytes, logger)
	if err != nil {
		t.Fatalf("create vmPool error: %v", err)
	}

	defer func() {
		vmPool.close()
	}()

	wrappedInstance, err := vmPool.NewInstance()
	if err != nil {
		t.Fatalf("vmPool.NewInstance() error: %v", err)
	}

	if wrappedInstance == nil {
		t.Fatalf("vmPool.NewInstance() return nil")
	}
	defer func() {
		vmPool.CloseInstance(wrappedInstance)
	}()
}

func TestGetWrappedInstance(t *testing.T) {

	wasmBytes, contractId, logger := prepareContract("./testdata/rust-counter-2.0.0.wasm", t)

	vmPool, err := newVmPool(&contractId, wasmBytes, logger)
	if err != nil {
		t.Fatalf("create vmPool error: %v", err)
	}
	defer vmPool.close()

	fmt.Printf("vmPool has %v instane.\n", vmPool.currentSize)

	waitTime := time.Second * 2
	var wg sync.WaitGroup
	for i := 0; i < 22; i++ {
		wg.Add(1)
		go func(no int) {
			defer wg.Done()

			wrappedInstance := vmPool.GetInstance()
			defer vmPool.RevertInstance(wrappedInstance)

			fmt.Printf("vmPool used %v instane.\n", vmPool.useCount)
			time.Sleep(waitTime)
		}(i)
	}
	wg.Wait()

	fmt.Printf("vmPool.getAverageDelay() = %v \n", vmPool.getAverageDelay())
	fmt.Printf("(%v * 10 + %v * 2)/22 = %v \n", waitTime.Milliseconds(), waitTime.Milliseconds()*2, (waitTime.Milliseconds()*10+waitTime.Milliseconds()*2*2)/22)
}

func TestGrowAndShrink(t *testing.T) {
	wasmBytes, contractId, logger := prepareContract("./testdata/rust-counter-2.0.0.wasm", t)

	vmPool, err := newVmPool(&contractId, wasmBytes, logger)
	if err != nil {
		t.Fatalf("create vmPool error: %v", err)
	}
	defer vmPool.close()

	assert.Equal(t, vmPool.currentSize, int32(0), "new vmPool should has zero instances.")

	// make averageDelay > 10 Millisecond
	waitTime := time.Millisecond * 50
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(no int) {
			defer wg.Done()

			wrappedInstance := vmPool.GetInstance()
			defer vmPool.RevertInstance(wrappedInstance)

			fmt.Printf("vmPool used %v instane, currentSize = %v \n", vmPool.useCount, vmPool.currentSize)
			time.Sleep(waitTime)
		}(i)
	}
	wg.Wait()
	fmt.Printf("vmPool.getAverageDelay() = %v \n", vmPool.getAverageDelay())

	assert.Equal(t, vmPool.currentSize, int32(10), "new vmPool should has 10 instances.")
	assert.Greater(t, vmPool.getAverageDelay(), int32(defaultDelayTolerance), "vmPool should got Delay longer than default.")

	// make 11 request => vmPool grow()
	var lock sync.Mutex
	var instancePool []*wrappedInstance
	for i := 0; i < 11; i++ {
		wg.Add(1)
		go func(no int) {
			defer wg.Done()

			fmt.Printf("GoRoutine(%v) try to get an instance \n", no)
			wrappedInstance := vmPool.GetInstance()
			fmt.Printf("GoRoutine(%v) got an instance \n", no)
			lock.Lock()
			instancePool = append(instancePool, wrappedInstance)
			lock.Unlock()
			fmt.Printf("vmPool used %v instane, currentSize = %v \n", vmPool.useCount, vmPool.currentSize)
			time.Sleep(time.Second)
		}(i)
	}
	wg.Wait()

	for i, wrappedInstance := range instancePool {
		fmt.Printf("revert %v instance, currentSize = %v \n", i, vmPool.currentSize)
		vmPool.RevertInstance(wrappedInstance)
	}
	assert.Equal(t, vmPool.currentSize, int32(20), "new vmPool should has 20 instances.")

	vmPool.shrink(defaultDelayTolerance)
	assert.Equal(t, vmPool.currentSize, int32(10), "new vmPool should has 10 instances after shrink.")

}
