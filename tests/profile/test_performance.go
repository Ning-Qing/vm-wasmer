/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io/ioutil"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	tests "chainmaker.org/chainmaker/vm-wasmer-test/v2"
)

var (
	ConfigFilePath   = "../config/wx-org1/chainmaker.yml"
	UserCertFilePath = "../config/crypto-config/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
)

func main() {

	contractName := "counter"
	contractVersion := "v1.0.0"
	wasmFilePath := "../../testdata/rust-counter-2.0.0.wasm"

	contract, vmManager, snapshot, wasmByteCode, err := tests.BeforeCmdExecute(
		tests.ChainId,
		contractName,
		contractVersion,
		wasmFilePath,
		ConfigFilePath,
		UserCertFilePath)
	if err != nil {
		fmt.Printf("error occur: %v \n", err)
		return
	}
	blockchainStoreMock := snapshot.GetBlockchainStoreMock()

	// 放入调用合约的数据
	err = tests.PutContractIdIntoStore(blockchainStoreMock, contractName, contract)
	if err != nil {
		fmt.Printf("error occur: %v \n", err)
		return
	}
	tests.PutContractBytecodeIntoStore(blockchainStoreMock, contractName, wasmByteCode)
	// 放入被调用合约的数据
	calleeContract := commonPb.Contract{
		Name:        "contract01",
		Version:     "v1.0.0",
		RuntimeType: commonPb.RuntimeType_WASMER,
		Creator:     contract.Creator,
	}
	err = tests.PutContractIdIntoStore(blockchainStoreMock, "contract01", &calleeContract)
	if err != nil {
		fmt.Printf("error occur: %v \n", err)
		return
	}
	// 跨合约调用测试中写死了合约名称为'contract01'
	tests.PutContractBytecodeIntoStore(blockchainStoreMock, "contract01", wasmByteCode)

	testCallMultiTimes(contract, vmManager, snapshot, wasmByteCode)

}

func testCallMultiTimes(pbContract *commonPb.Contract,
	vmManager protocol.VmManager,
	snapshotMock *tests.SnapshotMock,
	wasmByteCode []byte) {

	blockchainStore := snapshotMock.GetBlockchainStoreMock()

	// 读取用户证书
	userCertBytes, err := ioutil.ReadFile(UserCertFilePath)
	if err != nil {
		fmt.Printf("#### read '%v' error: %v \n", UserCertFilePath, err)
	}

	//init pbContract
	fmt.Println("> > > > > > > >  call Contract ==> init_context()  > > > > > > > >")
	args := make(map[string][]byte)

	tests.InvokeContractWithParameters("init_contract", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()

	fmt.Println()
	time.Sleep(time.Second * 10)
	start := time.Now().UnixNano()
	totalCallTimes := int64(2000000)
	vmGoroutineNum := int64(100)
	finishNum := int64(0)
	for i := int64(0); i < totalCallTimes; {
		var createNum int64
		if i+vmGoroutineNum >= totalCallTimes {
			createNum = totalCallTimes - i
		} else {
			createNum = vmGoroutineNum
		}
		i += createNum

		wg := sync.WaitGroup{}
		for j := int64(0); j < createNum; j++ {
			wg.Add(1)
			go func(goRoutineNum int64) {
				defer func() {
					wg.Done()
					//fmt.Printf("goroutine %v exit. \n", goRoutineNum)
				}()

				args := make(map[string][]byte)
				args["field"] = []byte(fmt.Sprintf("field_%v", j))
				args["value"] = []byte(fmt.Sprintf("value_%v", j))

				tests.InvokeContractWithParameters("increase", pbContract, wasmByteCode, vmManager, snapshotMock,
					args, userCertBytes)
				atomic.AddInt64(&finishNum, 1)
			}(j)
		}
		wg.Wait()
		end := time.Now().UnixNano()
		fmt.Printf("finished %d task in %ds, average tps is %d, totalCallTimes: %d, vmGoroutineNum: %d, createNum: %d, i: %d. \n",
			finishNum, end-start, finishNum*1e9/(end-start), totalCallTimes, vmGoroutineNum, createNum, i)
	}

	end := time.Now().UnixNano()
	//fmt.Printf("测试开始时间: \t %v \n", testBegin.Format("2006-01-02 15:04:05"))
	//fmt.Printf("测试结束时间: \t %v \n", testEnd.Format("2006-01-02 15:04:05"))
	fmt.Printf("平均处理性能: \t %v 每秒 \n", finishNum*1e9/(end-start))
	debug.FreeOSMemory()
	time.Sleep(time.Second * 300000)
	fmt.Println("<= call tests end")
}
