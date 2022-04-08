/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/utils/v2"
	wasmergo "github.com/Ning-Qing/vm-wasmer/v2/wasmer-go"
)

const (
	// refresh vmPool time, use for grow or shrink
	defaultRefreshTime = time.Hour * 12
	// the max pool size for every contract
	defaultMaxSize = 1000
	// the min pool size
	defaultMinSize = 10
	// grow pool size
	defaultChangeSize = 10
	// if get instance avg time greater than this value, should grow pool, Millisecond as unit
	defaultDelayTolerance = 10
	// if apply times greater than this value, should grow pool
	defaultApplyThreshold = 100
	// if wasmer instance invoke error more than N times, should close and discard this instance
	defaultDiscardCount = 10
)

// vmPool, each contract has a vm pool providing multiple vm instances to call
// vm pool can grow and shrink on demand
type vmPool struct {
	// the corresponding contract info
	contractId *commonPb.Contract
	byteCode   []byte
	store      *wasmergo.Store
	module     *wasmergo.Module
	// wasmergo instance pool
	instances chan *wrappedInstance
	// current instance size in pool
	currentSize int32
	// use count from last refresh
	useCount int32
	// total delay (in ms) from last refresh
	totalDelay int32
	// total application count for pool grow
	// if we cannot get instance right now, applyGrowCount++
	applyGrowCount int32
	// apply signal channel
	applySignalC    chan struct{}
	closeC          chan struct{}
	resetC          chan struct{}
	removeInstanceC chan struct{}
	addInstanceC    chan struct{}
	log             *logger.CMLogger
}

// wrappedInstance wraps instance with id and other info
type wrappedInstance struct {
	// id
	id string
	// wasmergo instance provided by wasmer
	wasmInstance *wasmergo.Instance
	// lastUseTime, unix timestamp in ms
	lastUseTime int64
	// createTime, unix timestamp in ms
	createTime int64
	// errCount, current instance invoke method error count
	errCount int32
}

// GetInstance get a vm instance to run contract
// should be followed by defer resetInstance
func (p *vmPool) GetInstance() *wrappedInstance {

	var instance *wrappedInstance
	// get instance from vm pool
	select {
	case instance = <-p.instances:
		// concurrency safe here
		atomic.AddInt32(&p.useCount, 1)
		instance.lastUseTime = utils.CurrentTimeMillisSeconds()
		return instance
	default:
		// nothing
	}
	if instance == nil {
		log.Debugf("can't get wrappedInstance from vmPool.")
	}

	// if we cannot get it right now, send apply signal and wait
	// add wait time to total delay
	curTimeMS1 := utils.CurrentTimeMillisSeconds()
	go func() {
		p.applySignalC <- struct{}{}
		log.Debugf("send 'applySignal' to vmPool.")
	}()

	instance = <-p.instances
	log.Debugf("got an wrappedInstance from vmPool.")
	atomic.AddInt32(&p.useCount, 1)
	curTimeMS2 := utils.CurrentTimeMillisSeconds()
	instance.lastUseTime = curTimeMS2
	elapsedTimeMS := int32(curTimeMS2 - curTimeMS1)
	atomic.AddInt32(&p.totalDelay, elapsedTimeMS)

	return instance
}

// RevertInstance revert instance to pool
func (p *vmPool) RevertInstance(instance *wrappedInstance) {
	if p.shouldDiscard(instance) {
		go func() {
			p.removeInstanceC <- struct{}{}
			p.addInstanceC <- struct{}{}
			p.CloseInstance(instance)
		}()
	} else {
		p.instances <- instance
	}
}

// NewInstance create a wasmer instance directly, for cross contract call
func (p *vmPool) NewInstance() (*wrappedInstance, error) {
	return p.newInstanceFromModule()
}

// CloseInstance close a wasmer instance directly, for cross contract call
func (p *vmPool) CloseInstance(instance *wrappedInstance) {
	if instance != nil {
		if err := CallDeallocate(instance.wasmInstance); err != nil {
			p.log.Errorf("CallDeallocate(...) error: %v", err)
		}
		instance.wasmInstance.Close()
		instance = nil
	}
}

func newVmPool(contractId *commonPb.Contract, byteCode []byte, log *logger.CMLogger) (*vmPool, error) {
	store := wasmergo.NewStore(wasmergo.NewUniversalEngine())
	if err := wasmergo.ValidateModule(store, byteCode); err != nil {
		return nil, fmt.Errorf("[%s_%s], byte code validation failed, err = %v", contractId.Name, contractId.Version, err)
	}

	module, err := wasmergo.NewModule(store, byteCode)
	if err != nil {
		return nil, fmt.Errorf("[%s_%s], byte code compile failed", contractId.Name, contractId.Version)
	}

	vmPool := &vmPool{
		contractId:      contractId,
		byteCode:        byteCode,
		store:           store,
		module:          module,
		instances:       make(chan *wrappedInstance, defaultMaxSize),
		currentSize:     0,
		useCount:        0,
		totalDelay:      0,
		applyGrowCount:  0,
		applySignalC:    make(chan struct{}),
		removeInstanceC: make(chan struct{}),
		addInstanceC:    make(chan struct{}),
		closeC:          make(chan struct{}),
		resetC:          make(chan struct{}),
		log:             log,
	}

	instance, err := vmPool.newInstanceFromModule()
	if err != nil {
		return nil, fmt.Errorf("[%s_%s], byte code compile failed, %s", contractId.Name, contractId.Version, err.Error())
	}

	instance.wasmInstance.Close()
	log.Infof("vm pool verify byteCode finish.")

	go vmPool.startRefreshingLoop()
	log.Infof("vm pool startRefreshingLoop...")
	return vmPool, nil
}

// startRefreshingLoop refreshing loop manages the vm pool
// all grow and shrink operations are called here
func (p *vmPool) startRefreshingLoop() {

	refreshTimer := time.NewTimer(defaultRefreshTime)
	key := p.contractId.Name + "_" + p.contractId.Version
	for {
		select {
		case <-p.applySignalC:
			log.Debug("vmPool handling an applySignal")
			p.applyGrowCount++
			if p.shouldGrow() {
				log.Debugf("vmPool should grow %v wrappedInstance.", defaultChangeSize)
				p.grow(defaultChangeSize)
				p.applyGrowCount = 0
				p.log.Infof("[%s] vm pool grows by %d, the current size is %d",
					key, defaultChangeSize, p.currentSize)
			}
		case <-refreshTimer.C:
			p.log.Debugf("[%s] vm pool refresh timer expires. current size is %d, delay is %dms",
				key, p.currentSize, p.getAverageDelay())
			if p.shouldGrow() {
				p.grow(defaultChangeSize)
				p.applyGrowCount = 0
				p.log.Infof("[%s] vm pool grows by %d, the current size is %d",
					key, defaultChangeSize, p.currentSize)
			} else if p.shouldShrink() {
				p.shrink(defaultChangeSize)
				p.log.Infof("[%s] vm pool shrinks by %d, the current size is %d",
					key, defaultChangeSize, p.currentSize)
			}

			// other go routine may modify useCount & totalDelay
			// so we use atomic operation here
			atomic.StoreInt32(&p.useCount, 0)
			atomic.StoreInt32(&p.totalDelay, 0)
			refreshTimer.Reset(defaultRefreshTime)
		case <-p.closeC:
			refreshTimer.Stop()
			for p.currentSize > 0 {
				instance := <-p.instances
				if err := CallDeallocate(instance.wasmInstance); err != nil {
					p.log.Errorf("CallDeallocate(...) error: %v", err)
				}
				instance.wasmInstance.Close()
				p.currentSize--
			}
			close(p.instances)
			return
		case <-p.resetC:
			for p.currentSize > 0 {
				instance := <-p.instances
				if err := CallDeallocate(instance.wasmInstance); err != nil {
					p.log.Errorf("CallDeallocate(...) error: %v", err)
				}
				instance.wasmInstance.Close()
				p.currentSize--
			}
			close(p.instances)
			p.instances = make(chan *wrappedInstance, defaultMaxSize)
			p.grow(defaultMinSize)
		case <-p.removeInstanceC:
			p.currentSize--
		case <-p.addInstanceC:
			p.grow(1)
		}
	}
}

// shouldGrow grow vm pool when
// 1. current size + grow size <= max size, AND
// 2.1. apply count >= apply threshold, OR
// 2.2. average delay > delay tolerance (int operation here is safe)
func (p *vmPool) shouldGrow() bool {
	fmt.Println("1")
	if p.currentSize < defaultMinSize {
		fmt.Println("2")
		return true
	}
	fmt.Printf("3 currentSize = %v \n", p.currentSize)
	if p.currentSize+defaultChangeSize <= defaultMaxSize {
		if p.applyGrowCount > defaultApplyThreshold {
			fmt.Println("3")
			return true
		}

		fmt.Printf("4 average = %v \n", p.getAverageDelay())
		if p.getAverageDelay() > int32(defaultDelayTolerance) {
			return true
		}

		if p.currentSize < int32(defaultMinSize) {
			return true
		}
	}
	return false
}

func (p *vmPool) grow(count int32) {
	for count > 0 {
		size := int32(defaultChangeSize)
		if count < size {
			size = count
		}
		count -= size

		wg := sync.WaitGroup{}
		for i := int32(0); i < size; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				instance, _ := p.newInstanceFromModule()
				p.instances <- instance
				atomic.AddInt32(&p.currentSize, 1)
			}()
		}
		wg.Wait()
		p.log.Infof("vm pool grow size = %d", size)
	}
}

// shouldShrink shrink vm pool when
// 1. current size > min size, AND
// 2. average delay <= delay tolerance (int operation here is safe)
func (p *vmPool) shouldShrink() bool {
	if p.currentSize > defaultMinSize && p.getAverageDelay() <=
		int32(defaultDelayTolerance) && p.currentSize > defaultChangeSize {
		return true
	}
	return false
}

func (p *vmPool) shrink(count int32) {
	for i := int32(0); i < count; i++ {
		instance := <-p.instances
		if err := CallDeallocate(instance.wasmInstance); err != nil {
			p.log.Errorf("CallDeallocate(...) error: %v", err)
		}
		instance.wasmInstance.Close()
		instance = nil
		p.currentSize--
	}
}

// shouldDiscard discard instance when
// error count times more than defaultDiscardCount
func (p *vmPool) shouldDiscard(instance *wrappedInstance) bool {
	return instance.errCount > defaultDiscardCount
}

func (p *vmPool) NewInstanceFromByteCode() (*wrappedInstance, error) {
	vb := GetVmBridgeManager()
	wasmInstance, err := vb.NewWasmInstance(p.store, p.byteCode)
	if err != nil {
		p.log.Errorf("newInstanceFromByteCode fail: %s", err.Error())
		return nil, err
	}

	instance := &wrappedInstance{
		id:           uuid.GetUUID(),
		wasmInstance: wasmInstance,
		lastUseTime:  utils.CurrentTimeMillisSeconds(),
		createTime:   utils.CurrentTimeMillisSeconds(),
		errCount:     0,
	}
	return instance, nil
}

func (p *vmPool) newInstanceFromModule() (*wrappedInstance, error) {
	vb := GetVmBridgeManager()
	env := CMEnvironment{
		instance: nil,
	}
	wasmInstance, err := wasmergo.NewInstance(p.module, vb.GetImports(p.store, &env))
	if err != nil {
		p.log.Errorf("newInstanceFromModule fail: %s", err.Error())
		return nil, err
	}
	env.instance = wasmInstance

	instance := &wrappedInstance{
		id:           uuid.GetUUID(),
		wasmInstance: wasmInstance,
		lastUseTime:  utils.CurrentTimeMillisSeconds(),
		createTime:   utils.CurrentTimeMillisSeconds(),
		errCount:     0,
	}
	return instance, nil
}

// getAverageDelay average delay calculation here maybe not so accurate due to concurrency
// but we can still use it to decide grow/shrink or not
func (p *vmPool) getAverageDelay() int32 {
	delay := atomic.LoadInt32(&p.totalDelay)
	count := atomic.LoadInt32(&p.useCount)
	if count == 0 {
		return 0
	}
	return delay / count
}

// reset the pool instances
func (p *vmPool) reset() {
	p.resetC <- struct{}{}
}

// close the pool
func (p *vmPool) close() {
	close(p.closeC)
}
