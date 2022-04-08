/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package wasmer

import (
	"strconv"
	"sync"
	"unsafe"

	"chainmaker.org/chainmaker/store/v2/types"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm/v2"

	"github.com/Ning-Qing/vm-wasmer/v2/wasmer-go"

	"chainmaker.org/chainmaker/common/v2/serialize"
	"chainmaker.org/chainmaker/protocol/v2"
)

// #include <stdlib.h>

// extern int sysCall(void *context, int requestHeaderPtr, int requestHeaderLen, int requestBodyPtr, int requestBodyLen);
// extern void logMessage(void *context, int pointer, int length);
// extern int fdWrite(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern int fdRead(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern int fdClose(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern int fdSeek(void *contextfd,int iovs,int iovsPtr ,int iovsLen,int nwrittenPtr);
// extern void procExit(void *contextfd,int exitCode);
import "C"

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

// logMessage print log to file
//export logMessage
func logMessage(context unsafe.Pointer, pointer int32, length int32) {
	var instanceContext = wasmer.IntoInstanceContext(context)
	var memory = instanceContext.Memory().Data()

	gotText := string(memory[pointer : pointer+length])
	log.Debugf("wasmer log>> " + gotText)
}

// sysCall wasmer vm call chain entry
//export sysCall
func sysCall(context unsafe.Pointer,
	requestHeaderPtr int32, requestHeaderLen int32,
	requestBodyPtr int32, requestBodyLen int32) int32 {

	if requestHeaderLen == 0 {
		log.Error("wasmer log>> requestHeader is null.")
		return protocol.ContractSdkSignalResultFail
	}

	// get memory
	instanceContext := wasmer.IntoInstanceContext(context)
	memory := instanceContext.Memory().Data()

	// get request header/body from memory
	requestHeaderBytes := make([]byte, requestHeaderLen)
	copy(requestHeaderBytes, memory[requestHeaderPtr:requestHeaderPtr+requestHeaderLen])
	requestBodyBytes := make([]byte, requestBodyLen)
	copy(requestBodyBytes, memory[requestBodyPtr:requestBodyPtr+requestBodyLen])
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
		Memory:      memory,
		ChainId:     simContext.ChainId,
	}

	log.Infof("### enter syscall handling, method = '%v'", method)
	var ret int32
	if ret = waciInstance.invoke(method); ret == protocol.ContractSdkSignalResultFail {
		log.Infof("invoke WaciInstance error: method = %v", method)
	}

	log.Debugf("### leave syscall handling, method = '%v'", method)

	return ret
}

//nolint
func (s *WaciInstance) invoke(method interface{}) int32 {
	log.Infof("sysCall() => '%s' method", method)
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
	result, gas, specialTxType, err := wacsi.CallContract(s.RequestBody, s.Sc.TxSimContext, s.Memory,
		s.Sc.GetStateCache, s.Sc.Instance.GetGasUsed(), isLen)
	if result == nil {
		s.Sc.GetStateCache = nil // reset data
		//s.Sc.ContractEvent = nil
	} else {
		s.Sc.GetStateCache = result.Result // reset data
		s.Sc.ContractEvent = append(s.Sc.ContractEvent, result.ContractEvent...)
	}
	s.Sc.Instance.SetGasUsed(gas)
	s.Sc.SpecialTxType = specialTxType
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
func fdWrite(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export fdRead
func fdRead(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export fdClose
func fdClose(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export fdSeek
func fdSeek(context unsafe.Pointer, fd int32, iovsPtr int32, iovsLen int32, nwrittenPtr int32) (err int32) {
	return protocol.ContractSdkSignalResultSuccess
}

//export procExit
func procExit(context unsafe.Pointer, exitCode int32) {
	panic("exit called by contract, code:" + strconv.Itoa(int(exitCode)))
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
func GetVmBridgeManager() *vmBridgeManager {
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
func (b *vmBridgeManager) NewWasmInstance(byteCode []byte) (wasmer.Instance, error) {
	return wasmer.NewInstanceWithImports(byteCode, b.GetImports())
}

// GetImports return export interface to cgo
func (b *vmBridgeManager) GetImports() *wasmer.Imports {
	var err error
	imports := wasmer.NewImports().Namespace("env")
	if _, err = imports.Append("sys_call", sysCall, C.sysCall); err != nil {
		panic("add 'sys_call' into Imports error")
	}
	// parameter explain:
	// 	1、["log_message"]: rust extern "C" method name
	//	2、[logMessage] go method ptr
	//	3、[C.logMessage] cgo function pointer.
	if _, err = imports.Append("log_message", logMessage, C.logMessage); err != nil {
		panic("add 'log_message' into Imports error")
	}
	// for wacsi empty interface
	imports.Namespace("wasi_unstable")

	if _, err = imports.Append("fd_write", fdWrite, C.fdWrite); err != nil {
		panic("add 'fd_write' into Imports error")
	}
	if _, err = imports.Append("fd_read", fdRead, C.fdRead); err != nil {
		panic("add 'fd_read' into Imports error")
	}
	if _, err = imports.Append("fd_close", fdClose, C.fdClose); err != nil {
		panic("add 'fd_close' into Imports error")
	}
	if _, err = imports.Append("fd_seek", fdSeek, C.fdSeek); err != nil {
		panic("add 'fd_seek' into Imports error")
	}

	imports.Namespace("wasi_snapshot_preview1")
	if _, err = imports.Append("proc_exit", procExit, C.procExit); err != nil {
		panic("add 'proc_exit' into Imports error")
	}
	//imports.Append("fd_write", fdWrite2, C.fdWrite2)
	//imports.Append("environ_sizes_get", fdWrite, C.fdWrite)
	//imports.Append("proc_exit", fdWrite, C.fdWrite)
	//imports.Append("environ_get", fdWrite, C.fdWrite)

	return imports
}
