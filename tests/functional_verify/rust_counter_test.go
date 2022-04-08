/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package functional_verify

import (
	tests "chainmaker.org/chainmaker/vm-wasmer-test/v2"
	"fmt"
	"io/ioutil"
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

func TestCounter2(t *testing.T) {

	contractName := "counter"
	contractVersion := "v1.0.0"
	wasmFilePath := "../../testdata/rust-counter-2.0.0.wasm"

	contract, vmManager, snapshotMock, wasmByteCode, err := tests.BeforeCmdExecute(tests.ChainId, contractName,
		contractVersion, wasmFilePath, ConfigFilePath, UserCertFilePath)
	if err != nil {
		t.Fatal(err)
	}
	blockchainStoreMock := snapshotMock.GetBlockchainStoreMock()

	err = tests.PutContractIdIntoStore(blockchainStoreMock, contractName, contract)
	if err != nil {
		t.Fatal(err)
	}
	tests.PutContractBytecodeIntoStore(blockchainStoreMock, contractName, wasmByteCode)

	testCounterContract(contract, vmManager, snapshotMock, wasmByteCode)
}

func testCounterContract(pbContract *commonPb.Contract,
	vmManager protocol.VmManager,
	snapshot protocol.Snapshot,
	wasmBytecode []byte) {

	// 读取用户证书
	userCertBytes, err := ioutil.ReadFile(UserCertFilePath)
	if err != nil {
		fmt.Printf("#### read '%v' error: %v \n", UserCertFilePath, err)
	}

	parameters := make(map[string][]byte)

	tests.InvokeContractWithParameters("init_contract", pbContract, wasmBytecode, vmManager, snapshot,
		parameters, userCertBytes)

	tests.InvokeContractWithParameters("increase", pbContract, wasmBytecode, vmManager, snapshot,
		parameters, userCertBytes)

	tests.InvokeContractWithParameters("increase", pbContract, wasmBytecode, vmManager, snapshot,
		parameters, userCertBytes)

	tests.InvokeContractWithParameters("increase", pbContract, wasmBytecode, vmManager, snapshot,
		parameters, userCertBytes)

	tests.InvokeContractWithParameters("query", pbContract, wasmBytecode, vmManager, snapshot,
		parameters, userCertBytes)
}
