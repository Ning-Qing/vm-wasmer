/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"testing"
)

func TestNewRuntimeInstance(t *testing.T) {

	chainId := "chain02"
	blockVersion := uint32(1)
	contractName := "ContractCounter"
	method := "increase"

	wasmBytes, contractId, logger := prepareContract("./testdata/rust-counter-2.0.0.wasm", t)

	parameters := make(map[string][]byte)
	parameters["key"] = []byte("test_key")
	fillingBaseParams(parameters)

	txSimContext := prepareTxSimContext(chainId, blockVersion, contractName, method, parameters, SnapshotMock{})

	manager := NewInstancesManager(chainId)
	runtimeInst, err := manager.NewRuntimeInstance(
		nil, "", "", "", // parameters is no use for wasmer
		&contractId, wasmBytes, logger)
	if err != nil {
		t.Fatalf("NewRuntimeInstance() error: %v", err)
	}

	result, _ := runtimeInst.Invoke(&contractId, method, wasmBytes, parameters, txSimContext, 0)
	fmt.Printf("1) execute result = %v", result)

	result, _ = runtimeInst.Invoke(&contractId, method, wasmBytes, parameters, txSimContext, 0)
	fmt.Printf("2) execute result = %v", result)
}
