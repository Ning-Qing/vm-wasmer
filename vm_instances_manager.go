/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

// InstancesManager manages vm pools for all contracts
type InstancesManager struct {
	// chain identifier
	chainId string
	// control map operations
	m sync.RWMutex
	// contractName_contractVersion -> vm pool
	instanceMap map[string]*vmPool
	// module log
	log *logger.CMLogger
}

// NewInstancesManager return InstancesManager for every chain
func NewInstancesManager(chainId string) *InstancesManager {
	vmPoolManager := &InstancesManager{
		instanceMap: make(map[string]*vmPool),
		log:         logger.GetLoggerByChain(logger.MODULE_VM, chainId),
		chainId:     chainId,
	}
	return vmPoolManager
}

// NewRuntimeInstance init vm pool and check byteCode correctness
func (m *InstancesManager) NewRuntimeInstance(
	txSimContext protocol.TxSimContext,
	chainId, method, codePath string,
	contract *commonPb.Contract,
	byteCode []byte,
	log protocol.Logger) (protocol.RuntimeInstance, error) {
	var err error
	if contract == nil || contract.Name == "" || contract.Version == "" {
		err = fmt.Errorf("contract id is nil")
		m.log.Warn(err)
		return nil, err
	}

	if len(byteCode) == 0 {
		err = fmt.Errorf("[%s_%s], byte code is nil", contract.Name, contract.Version)
		m.log.Warn(err)
		return nil, err
	}

	pool, err := m.getVmPool(contract, byteCode)
	if err != nil || pool == nil {
		return nil, err
	}

	runtime := &RuntimeInstance{
		pool:             pool,
		log:              m.log,
		chainId:          m.chainId,
		instancesManager: m,
	}

	return runtime, nil
}

func (m *InstancesManager) getVmPool(contractId *commonPb.Contract, byteCode []byte) (*vmPool, error) {
	var err error
	key := contractId.Name + "_" + contractId.Version

	m.m.RLock()
	pool, ok := m.instanceMap[key]
	m.m.RUnlock()
	if !ok {
		m.m.Lock()
		defer m.m.Unlock()

		pool, ok = m.instanceMap[key]
		if !ok {
			start := utils.CurrentTimeMillisSeconds()
			m.log.Infof("[%s] init vm pool start", key)

			pool, err = newVmPool(contractId, byteCode, m.log)
			if err != nil {
				return nil, err
			}

			pool.grow(defaultMinSize)
			m.instanceMap[key] = pool
			end := utils.CurrentTimeMillisSeconds()
			m.log.Infof("[%s] init vmPool done, currentSize=%d, spend %dms", key, pool.currentSize, end-start)
		}
	}
	return pool, err
}

// close the contract vm pool
func (m *InstancesManager) CloseAVmPool(contractId *commonPb.Contract) {
	m.m.Lock()
	defer m.m.Unlock()

	key := contractId.Name + "_" + contractId.Version
	pool, ok := m.instanceMap[key]
	if ok {
		m.log.Infof("close pool %s", key)
		pool.close()
		delete(m.instanceMap, key)
	}
}

// close all contract vm pool
func (m *InstancesManager) CloseAllVmPool() {
	m.m.Lock()
	defer m.m.Unlock()

	for key, pool := range m.instanceMap {
		m.log.Infof("close pool %s", key)
		pool.close()
	}
	m.instanceMap = make(map[string]*vmPool)
}

// FIXME: 确认函数名是否多了字符A？@taifu
// reset a contract vm pool install
func (m *InstancesManager) ResetAVmPool(contractId *commonPb.Contract) {
	m.m.Lock()
	defer m.m.Unlock()

	key := contractId.Name + "_" + contractId.Version
	pool, ok := m.instanceMap[key]
	if ok {
		m.log.Infof("reset pool %s", key)
		pool.reset()
	}
}

// reset all contract pool instance
func (m *InstancesManager) ResetAllPool() {
	m.m.Lock()
	defer m.m.Unlock()

	for key, pool := range m.instanceMap {
		m.log.Infof("reset pool %s", key)
		pool.reset()
	}
}

func (m *InstancesManager) StartVM() error {
	return nil
}

func (m *InstancesManager) StopVM() error {
	return nil
}
