/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"strconv"
	"sync"

	"chainmaker.org/chainmaker/logger/v2"

	"chainmaker.org/chainmaker/common/v2/serialize"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/Ning-Qing/vm-wasmer/wasmer-go"
)

// SimContext record the contract context
type SimContext struct {
	TxSimContext   protocol.TxSimContext
	Contract       *commonPb.Contract
	ContractResult *commonPb.ContractResult
	Log            *logger.CMLogger
	Instance       *wasmer.Instance

	method        string
	parameters    map[string][]byte
	CtxPtr        int32
	GetStateCache []byte // cache call method GetStateLen value result, one cache per transaction
	ChainId       string
	ContractEvent []*commonPb.ContractEvent
	SpecialTxType protocol.ExecOrderTxType
}

// NewSimContext for every transaction
func NewSimContext(method string, log *logger.CMLogger, chainId string) *SimContext {
	sc := SimContext{
		method:  method,
		Log:     log,
		ChainId: chainId,
	}

	sc.putCtxPointer()

	return &sc
}

// CallMethod will call contract method
func (sc *SimContext) CallMethod(instance *wasmer.Instance) error {
	var bytes []byte

	runtimeFn, err := instance.Exports.GetRawFunction(protocol.ContractRuntimeTypeMethod)
	if err != nil {
		return fmt.Errorf("method [%s] not export, err = %v", protocol.ContractRuntimeTypeMethod, err)
	}
	defer runtimeFn.Close()

	sdkType, err := runtimeFn.Call()
	if err != nil {
		return err
	}

	runtimeSdkType, ok := sdkType.(int32)
	if !ok {
		return fmt.Errorf("sdkType is not int32 type")
	}
	if int32(commonPb.RuntimeType_WASMER) == runtimeSdkType {
		sc.parameters[protocol.ContractContextPtrParam] = []byte(strconv.Itoa(int(sc.CtxPtr)))
		ec := serialize.NewEasyCodecWithMap(sc.parameters)
		bytes = ec.Marshal()
	} else {
		return fmt.Errorf("runtime type error, expect rust:[%d], but got %d",
			uint64(commonPb.RuntimeType_WASMER), runtimeSdkType)
	}

	return sc.callContract(instance, sc.method, bytes)
}

func (sc *SimContext) callContract(instance *wasmer.Instance, methodName string, bytes []byte) error {

	lengthOfSubject := len(bytes)

	allocateFunc, err := instance.Exports.GetRawFunction(protocol.ContractAllocateMethod)
	if err != nil {
		return fmt.Errorf("method [%s] not export, err = %v", protocol.ContractAllocateMethod, err)
	}
	defer allocateFunc.Close()

	// Allocate memory for the subject, and get a pointer to it.
	allocateResult, err := allocateFunc.Call(lengthOfSubject)
	if err != nil {
		sc.Log.Errorf("contract invoke %s failed, %s", protocol.ContractAllocateMethod, err.Error())
		return fmt.Errorf("%s invoke failed. There may not be enough memory or CPU", protocol.ContractAllocateMethod)
	}
	dataPtr, ok := allocateResult.(int32)
	if !ok {
		return fmt.Errorf("allocateResult is not int32 type")
	}

	// Write the subject into the memory.
	exportMemory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return fmt.Errorf("[%s] can't get exported memory, err = %v", protocol.ContractAllocateMethod, err)
	}
	memory := exportMemory.Data()[dataPtr:]

	//copy(memory, bytes)
	for nth := 0; nth < lengthOfSubject; nth++ {
		memory[nth] = bytes[nth]
	}

	// fmt.Println("[sim_context.go]: callContract(...) was running...")
	// Calls the `invoke` exported function. Given the pointer to the subject.
	exportFunc, err := instance.Exports.GetRawFunction(methodName)
	if err != nil {
		return fmt.Errorf("find method [%s] failed, err = %v", methodName, err)
	}
	defer exportFunc.Close()

	_, err = exportFunc.Call()
	//fmt.Println("[sim_context.go]: callContract(...) finished.")
	if err != nil {
		return err
	}

	return err
}

// CallDeallocate deallocate vm memory before closing the instance
func CallDeallocate(instance *wasmer.Instance) error {
	instance.SetGasLimit(protocol.GasLimit)
	deallocFunc, err := instance.Exports.GetFunction(protocol.ContractDeallocateMethod)
	if err != nil {
		return err
	}
	_, err = deallocFunc(0)
	return err
}

// putCtxPointer revmoe SimContext from cache
func (sc *SimContext) removeCtxPointer() {
	vbm := GetVmBridgeManager()
	vbm.remove(sc.CtxPtr)
}

var ctxIndex = int32(0)
var lock sync.Mutex

// putCtxPointer save SimContext to cache
func (sc *SimContext) putCtxPointer() {
	lock.Lock()
	ctxIndex++
	if ctxIndex > 1e8 {
		ctxIndex = 0
	}
	sc.CtxPtr = ctxIndex
	lock.Unlock()
	vbm := GetVmBridgeManager()
	vbm.put(sc.CtxPtr, sc)
}
