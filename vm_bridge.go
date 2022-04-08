/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package wasmer

import (
	"errors"
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/store/v2/types"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm/v2"

	"github.com/Ning-Qing/vm-wasmer/v2/wasmer-go"

	"chainmaker.org/chainmaker/common/v2/serialize"
	"chainmaker.org/chainmaker/protocol/v2"
)

var log = logger.GetLogger(logger.MODULE_VM)

// Wacsi WebAssembly chainmaker system interface
var wacsi = vm.NewWacsi(log, &types.StandardSqlVerify{})

// WaciInstance record wasmer vm request parameter
type WaciInstance struct {
	Sc          *SimContext
	RequestBody []byte // sdk request param
	Memory      []byte // vm memory
	ChainId     string
}

// LogMessage print log to file
func (s *WaciInstance) LogMessage() int32 {
	s.Sc.Log.Debugf("wasmer log>> [%s] %s", s.Sc.TxSimContext.GetTx().Payload.TxId, string(s.RequestBody))
	return protocol.ContractSdkSignalResultSuccess
}

type CMEnvironment struct {
	instance *wasmer.Instance
}

// logMessage print log to certFile
//export logMessage
func logMessage(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	env, ok := environment.(*CMEnvironment)
	if !ok {
		return nil, fmt.Errorf("args 'environment' is not *CMEnvironment type")
	}
	instance := env.instance
	if instance == nil {
		return nil, errors.New("instance at Environment is nil")
	}
	exportMemory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return nil, err
	}

	pointer := args[0].I32()
	length := args[1].I32()
	gotText := string(exportMemory.Data()[pointer : pointer+length])
	log.Debug("wasmer log>> " + gotText)

	return []wasmer.Value{}, nil
}

// sysCall wasmer vm call chain entry
//export sysCall
func sysCall(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {

	env, ok := environment.(*CMEnvironment)
	if !ok {
		return nil, errors.New("args 'environment' is not *CMEnvironment type")
	}
	instance := env.instance
	if instance == nil {
		return nil, errors.New("instance at Environment is nil")
	}
	exportMemory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		return nil, err
	}

	requestHeaderPtr := args[0].I32()
	requestHeaderLen := args[1].I32()
	requestBodyPtr := args[2].I32()
	requestBodyLen := args[3].I32()
	if requestHeaderLen == 0 {
		log.Error("wasmer-go log >> requestHeader is null.")
		return []wasmer.Value{
			wasmer.NewValue(protocol.ContractSdkSignalResultFail, wasmer.I32),
		}, nil
	}

	// get request header/body from memory
	requestHeaderBytes := make([]byte, requestHeaderLen)
	copy(requestHeaderBytes, exportMemory.Data()[requestHeaderPtr:requestHeaderPtr+requestHeaderLen])
	requestBodyBytes := make([]byte, requestBodyLen)
	copy(requestBodyBytes, exportMemory.Data()[requestBodyPtr:requestBodyPtr+requestBodyLen])
	requestHeader := serialize.NewEasyCodecWithBytes(requestHeaderBytes)

	// get SimContext number from request header
	ctxPtr, err := requestHeader.GetValue("ctx_ptr", serialize.EasyKeyType_SYSTEM)
	if err != nil {
		log.Error("get ctx_ptr failed:%s requestHeader=%s requestBody=%s", "request header have no ctx_ptr",
			string(requestHeaderBytes), string(requestBodyBytes), err)
	}
	// get sys_call method from request header
	method, err := requestHeader.GetValue("method", serialize.EasyKeyType_SYSTEM)
	if err != nil {
		log.Error("get method failed:%s requestHeader=%s requestBody=%s", "request header have no method",
			string(requestHeaderBytes), string(requestBodyBytes), err)
	}

	vbm := GetVmBridgeManager()
	simContext := vbm.get(ctxPtr.(int32))

	// create new WaciInstance for operate on blockchain
	waciInstance := &WaciInstance{
		Sc:          simContext,
		RequestBody: requestBodyBytes,
		Memory:      exportMemory.Data(),
		ChainId:     simContext.ChainId,
	}

	log.Debugf("### enter syscall handling, method = '%v'", method)
	var ret int32
	if ret = waciInstance.invoke(method); ret == protocol.ContractSdkSignalResultFail {
		log.Infof("invoke WaciInstance error: method = %v", method)
	}

	log.Debugf("### leave syscall handling, method = '%v'", method)

	return []wasmer.Value{
		wasmer.NewValue(ret, wasmer.I32),
	}, nil
}

//nolint
func (s *WaciInstance) invoke(method interface{}) int32 {
	switch method.(string) {
	// common
	case protocol.ContractMethodLogMessage:
		return s.LogMessage()
	case protocol.ContractMethodSuccessResult:
		return s.SuccessResult()
	case protocol.ContractMethodErrorResult:
		return s.ErrorResult()
	case protocol.ContractMethodCallContract:
		return s.CallContract()
	case protocol.ContractMethodCallContractLen:
		return s.CallContractLen()
	case protocol.ContractMethodEmitEvent:
		return s.EmitEvent()
		// paillier
	case protocol.ContractMethodGetPaillierOperationResultLen:
		return s.GetPaillierResultLen()
	case protocol.ContractMethodGetPaillierOperationResult:
		return s.GetPaillierResult()
		// bulletproofs
	case protocol.ContractMethodGetBulletproofsResultLen:
		return s.GetBulletProofsResultLen()
	case protocol.ContractMethodGetBulletproofsResult:
		return s.GetBulletProofsResult()
	// kv
	case protocol.ContractMethodGetStateLen:
		return s.GetStateLen()
	case protocol.ContractMethodGetState:
		return s.GetState()
	case protocol.ContractMethodPutState:
		return s.PutState()
	case protocol.ContractMethodDeleteState:
		return s.DeleteState()
	case protocol.ContractMethodKvIterator:
		s.Sc.SpecialTxType = protocol.ExecOrderTxTypeIterator
		return s.KvIterator()
	case protocol.ContractMethodKvPreIterator:
		s.Sc.SpecialTxType = protocol.ExecOrderTxTypeIterator
		return s.KvPreIterator()
	case protocol.ContractMethodKvIteratorHasNext:
		return s.KvIteratorHasNext()
	case protocol.ContractMethodKvIteratorNextLen:
		return s.KvIteratorNextLen()
	case protocol.ContractMethodKvIteratorNext:
		return s.KvIteratorNext()
	case protocol.ContractMethodKvIteratorClose:
		return s.KvIteratorClose()
	// sql
	case protocol.ContractMethodExecuteUpdate:
		return s.ExecuteUpdate()
	case protocol.ContractMethodExecuteDdl:
		return s.ExecuteDDL()
	case protocol.ContractMethodExecuteQuery:
		return s.ExecuteQuery()
	case protocol.ContractMethodExecuteQueryOne:
		return s.ExecuteQueryOne()
	case protocol.ContractMethodExecuteQueryOneLen:
		return s.ExecuteQueryOneLen()
	case protocol.ContractMethodRSHasNext:
		return s.RSHasNext()
	case protocol.ContractMethodRSNextLen:
		return s.RSNextLen()
	case protocol.ContractMethodRSNext:
		return s.RSNext()
	case protocol.ContractMethodRSClose:
		return s.RSClose()
	default:
		return protocol.ContractSdkSignalResultFail
	}
}

// SuccessResult record the results of contract execution success
func (s *WaciInstance) SuccessResult() int32 {
	return wacsi.SuccessResult(s.Sc.ContractResult, s.RequestBody)
}

// ErrorResult record the results of contract execution error
func (s *WaciInstance) ErrorResult() int32 {
	return wacsi.ErrorResult(s.Sc.ContractResult, s.RequestBody)
}

//  CallContractLen invoke cross contract calls, save result to cache and putout result length
func (s *WaciInstance) CallContractLen() int32 {
	return s.callContractCore(true)
}

//  CallContractLen get cross contract call result from cache
func (s *WaciInstance) CallContract() int32 {
	return s.callContractCore(false)
}

func (s *WaciInstance) callContractCore(isLen bool) int32 {
	gasUsed := protocol.GasLimit - s.Sc.Instance.GetGasRemaining()
	result, gas, specialTxType, err := wacsi.CallContract(s.RequestBody, s.Sc.TxSimContext, s.Memory,
		s.Sc.GetStateCache, gasUsed, isLen)
	if result == nil {
		s.Sc.GetStateCache = nil // reset data
		//s.Sc.ContractEvent = nil
	} else {
		s.Sc.GetStateCache = result.Result // reset data
		s.Sc.ContractEvent = append(s.Sc.ContractEvent, result.ContractEvent...)
	}
	s.Sc.SpecialTxType = specialTxType
	s.Sc.Instance.SetGasLimit(protocol.GasLimit - gas)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// EmitEvent emit event to chain
func (s *WaciInstance) EmitEvent() int32 {
	contractEvent, err := wacsi.EmitEvent(s.RequestBody, s.Sc.TxSimContext, s.Sc.Contract, s.Sc.Log)
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	s.Sc.ContractEvent = append(s.Sc.ContractEvent, contractEvent)
	return protocol.ContractSdkSignalResultSuccess
}

// GetBulletProofsResultLen get bulletproofs operation result length from chain
func (s *WaciInstance) GetBulletProofsResultLen() int32 {
	return s.getBulletProofsResultCore(true)
}

// GetBulletProofsResult get bulletproofs operation result from chain
func (s *WaciInstance) GetBulletProofsResult() int32 {
	return s.getBulletProofsResultCore(false)
}

func (s *WaciInstance) getBulletProofsResultCore(isLen bool) int32 {
	data, err := wacsi.BulletProofsOperation(s.RequestBody, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// GetPaillierResultLen get paillier operation result length from chain
func (s *WaciInstance) GetPaillierResultLen() int32 {
	return s.getPaillierResultCore(true)
}

// GetPaillierResult get paillier operation result from chain
func (s *WaciInstance) GetPaillierResult() int32 {
	return s.getPaillierResultCore(false)
}

func (s *WaciInstance) getPaillierResultCore(isLen bool) int32 {
	data, err := wacsi.PaillierOperation(s.RequestBody, s.Memory, s.Sc.GetStateCache, isLen)
	s.Sc.GetStateCache = data // reset data
	if err != nil {
		s.recordMsg(err.Error())
		return protocol.ContractSdkSignalResultFail
	}
	return protocol.ContractSdkSignalResultSuccess
}

// wasi
//export fdWrite
func fdWrite(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	return []wasmer.Value{
		wasmer.NewValue(protocol.ContractSdkSignalResultSuccess, wasmer.I32),
	}, nil
}

//export fdRead
func fdRead(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	return []wasmer.Value{
		wasmer.NewValue(protocol.ContractSdkSignalResultSuccess, wasmer.I32),
	}, nil
}

//export fdClose
func fdClose(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	return []wasmer.Value{
		wasmer.NewValue(protocol.ContractSdkSignalResultSuccess, wasmer.I32),
	}, nil
}

//export fdSeek
func fdSeek(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	return []wasmer.Value{
		wasmer.NewValue(protocol.ContractSdkSignalResultSuccess, wasmer.I32),
	}, nil
}

//export procExit
func procExit(environment interface{}, args []wasmer.Value) ([]wasmer.Value, error) {
	return []wasmer.Value{}, nil
}

func (s *WaciInstance) recordMsg(msg string) int32 {
	if len(s.Sc.ContractResult.Message) > 0 {
		s.Sc.ContractResult.Message += ". error message: " + msg
	} else {
		s.Sc.ContractResult.Message += "error message: " + msg
	}
	s.Sc.ContractResult.Code = 1
	s.Sc.Log.Errorf("wasmer log>> [%s] %s", s.Sc.Contract.Name, msg)
	return protocol.ContractSdkSignalResultFail
}

var (
	vmBridgeManagerMutex = &sync.Mutex{}
	bridgeSingleton      *vmBridgeManager
)

type vmBridgeManager struct {
	//wasmImports *wasm.Imports
	pointerLock     sync.Mutex
	simContextCache map[int32]*SimContext
}

// GetVmBridgeManager get singleton vmBridgeManager struct
func GetVmBridgeManager() *vmBridgeManager { //nolint
	if bridgeSingleton == nil {
		vmBridgeManagerMutex.Lock()
		defer vmBridgeManagerMutex.Unlock()
		if bridgeSingleton == nil {
			log.Debugf("init vmBridgeManager")
			bridgeSingleton = &vmBridgeManager{}
			bridgeSingleton.simContextCache = make(map[int32]*SimContext)
			//bridgeSingleton.wasmImports = bridgeSingleton.GetImports()
		}
	}
	return bridgeSingleton
}

// put the context
func (b *vmBridgeManager) put(k int32, v *SimContext) {
	b.pointerLock.Lock()
	defer b.pointerLock.Unlock()
	b.simContextCache[k] = v
}

// get the context
func (b *vmBridgeManager) get(k int32) *SimContext {
	b.pointerLock.Lock()
	defer b.pointerLock.Unlock()
	return b.simContextCache[k]
}

// remove the context
func (b *vmBridgeManager) remove(k int32) {
	b.pointerLock.Lock()
	defer b.pointerLock.Unlock()
	delete(b.simContextCache, k)
}

// NewWasmInstance new wasm instance. Apply for new memory.
func (b *vmBridgeManager) NewWasmInstance(store *wasmer.Store, byteCode []byte) (*wasmer.Instance, error) {
	module, err := wasmer.NewModule(store, byteCode)
	if err != nil {
		return nil, err
	}

	env := CMEnvironment{instance: nil}
	imports := b.GetImports(store, &env)
	instance, err := wasmer.NewInstance(module, imports)
	if err != nil {
		return nil, err
	}

	env.instance = instance

	return instance, err
}

// GetImports return export interface to cgo
func (b *vmBridgeManager) GetImports(store *wasmer.Store, env *CMEnvironment) *wasmer.ImportObject {
	imports := wasmer.NewImportObject()

	// syscall
	syscall := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32)),
		env,
		sysCall)
	// log_message
	logmessage := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes()),
		env,
		logMessage)

	imports.Register(
		"env",
		map[string]wasmer.IntoExtern{
			"sys_call":    syscall,
			"log_message": logmessage,
		})

	fdread := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32)),
		env,
		fdRead)

	fdwrite := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32)),
		env,
		fdWrite)

	fdclose := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32)),
		env,
		fdClose)

	fdseek := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(
			wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32),
			wasmer.NewValueTypes(wasmer.I32)),
		env,
		fdSeek)

	// for wacsi empty interface
	imports.Register(
		"wasi_unstable",
		map[string]wasmer.IntoExtern{
			"fd_read":  fdread,
			"fd_write": fdwrite,
			"fd_close": fdclose,
			"fd_seek":  fdseek,
		})

	procexit := wasmer.NewFunctionWithEnvironment(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		env,
		procExit)

	// for wasi_snapshot_preview1
	imports.Register(
		"wasi_snapshot_preview1",
		map[string]wasmer.IntoExtern{
			"proc_exit": procexit,
		})

	return imports
}
