/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"

	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

// RuntimeInstance wasm runtime
type RuntimeInstance struct {
	pool             *vmPool
	log              *logger.CMLogger
	chainId          string
	instancesManager *InstancesManager
}

func (r *RuntimeInstance) Pool() *vmPool {
	return r.pool
}

// Invoke contract by call vm, implement protocol.RuntimeInstance
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string, byteCode []byte,
	parameters map[string][]byte, txContext protocol.TxSimContext, gasUsed uint64) (
	contractResult *commonPb.ContractResult, specialTxType protocol.ExecOrderTxType) {

	r.log.Debugf("called invoke for tx:%s", txContext.GetTx().Payload.TxId)
	logStr := fmt.Sprintf("wasmer runtime invoke[%s]: ", txContext.GetTx().Payload.TxId)
	startTime := utils.CurrentTimeMillisSeconds()

	// set default return value
	contractResult = &commonPb.ContractResult{
		Code:    uint32(0),
		Result:  nil,
		Message: "",
	}
	specialTxType = protocol.ExecOrderTxTypeNormal

	var instanceInfo *wrappedInstance
	defer func() {
		endTime := utils.CurrentTimeMillisSeconds()
		logStr = fmt.Sprintf("%s used time %d", logStr, endTime-startTime)
		r.log.Debugf(logStr)
		panicErr := recover()
		if panicErr != nil {
			contractResult.Code = 1
			contractResult.Message = fmt.Sprint(panicErr)
			if instanceInfo != nil {
				instanceInfo.errCount++
			}
			specialTxType = protocol.ExecOrderTxTypeNormal
		}
	}()

	// if cross contract call, then new instance
	if txContext.GetDepth() > 0 {
		var err error
		instanceInfo, err = r.pool.NewInstance()
		defer r.pool.CloseInstance(instanceInfo)
		if err != nil {
			panic(err)
		}
	} else {
		r.log.Debugf("before get instance for tx: %s", txContext.GetTx().Payload.TxId)
		instanceInfo = r.pool.GetInstance()
		r.log.Debugf("after get instance for tx: %s", txContext.GetTx().Payload.TxId)
		defer r.pool.RevertInstance(instanceInfo)
	}

	instance := instanceInfo.wasmInstance
	instance.SetGasLimit(protocol.GasLimit - gasUsed)

	var sc = NewSimContext(method, r.log, r.chainId)
	defer sc.removeCtxPointer()
	sc.Contract = contract
	sc.TxSimContext = txContext
	sc.ContractResult = contractResult
	sc.parameters = parameters
	sc.Instance = instance
	sc.SpecialTxType = protocol.ExecOrderTxTypeNormal

	err := sc.CallMethod(instance)
	r.log.Debugf("contract invoke finished, tx:%s, call method err is %s",
		txContext.GetTx().Payload.TxId, err)
	if err != nil {
		r.log.Errorf("contract invoke failed, %s, tx: %s", err, txContext.GetTx().Payload.TxId)
	}
	specialTxType = sc.SpecialTxType

	// gas Log
	gas := protocol.GasLimit - instance.GetGasRemaining()
	if instance.GetGasRemaining() <= 0 {
		err = fmt.Errorf("contract invoke failed, out of gas %d/%d, tx: %s", gas, int64(protocol.GasLimit),
			txContext.GetTx().Payload.TxId)
	}
	logStr += fmt.Sprintf("used gas %d ", gas)
	contractResult.GasUsed = gas

	if err != nil {
		contractResult.Code = 1
		msg := fmt.Sprintf("contract invoke failed, %s, tx: %s", err.Error(), txContext.GetTx().Payload.TxId)
		r.log.Errorf(msg)
		contractResult.Message = msg
		if method == "init_contract" {
			r.instancesManager.CloseAVmPool(contract)
		} else {
			instanceInfo.errCount++
		}
		return
	}
	contractResult.ContractEvent = sc.ContractEvent
	contractResult.GasUsed = gas
	return
}
