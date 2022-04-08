/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"bytes"
	"fmt"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/assert"
)

func readWriteSet(txSimContext protocol.TxSimContext) ([]byte, error) {
	rwSet := txSimContext.GetTxRWSet(true)
	fmt.Printf("rwSet = %v \n", rwSet)

	var result []byte
	for _, w := range rwSet.TxWrites {
		if bytes.Equal(w.Key, []byte("count#test_key")) {
			result = w.Value
		}
	}
	if result == nil {
		return nil, fmt.Errorf("write set contain no 'count#test_key'")
	}

	return result, nil
}

func TestInvoke(t *testing.T) {

	wasmBytes, contractId, logger := prepareContract("./testdata/rust-counter-2.0.0.wasm", t)

	vmPool, err := newVmPool(&contractId, wasmBytes, logger)
	if err != nil {
		t.Fatalf("create vmPool error: %v", err)
	}

	defer func() {
		vmPool.close()
	}()

	runtimeInst := RuntimeInstance{
		pool:    vmPool,
		log:     logger,
		chainId: ChainId,
	}

	parameters := make(map[string][]byte)
	parameters["key"] = []byte("test_key")
	fillingBaseParams(parameters)

	// 测试一次调用结果是否正确
	txSimContext := prepareTxSimContext(ChainId, BlockVersion, ContractName, "increase", parameters, SnapshotMock{})
	runtimeInst.Invoke(&contractId, "increase", wasmBytes, parameters, txSimContext, 0)
	result, err := readWriteSet(txSimContext)
	if err != nil {
		t.Fatalf("read write set error: %v", err)
	}
	assert.Equal(t, result, []byte{1, 0, 0, 0}, "increase result not as expect.")

	// 测试第二次调用结果是否正确
	runtimeInst.Invoke(&contractId, "increase", wasmBytes, parameters, txSimContext, 0)
	result, err = readWriteSet(txSimContext)
	if err != nil {
		t.Fatalf("read write set error: %v", err)
	}
	assert.Equal(t, result, []byte{2, 0, 0, 0}, "increase result not as expect.")
}
