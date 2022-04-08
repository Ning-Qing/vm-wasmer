/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wasmer

import (
	"fmt"
	"io/ioutil"
	"testing"

	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	logger2 "chainmaker.org/chainmaker/logger/v2"
	accessPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm/v2"
)

const (
	ContractName    = "ContractTest001"
	ContractVersion = "1.0.0"
	ChainId         = "chain02"
	BlockVersion    = uint32(1)
)

func readWasmFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func prepareContract(filepath string, t *testing.T) ([]byte, commonPb.Contract, *logger2.CMLogger) {
	wasmBytes, err := readWasmFile(filepath)
	if err != nil {
		t.Fatalf("read wasm file error: %v", err)
	}

	contractId := commonPb.Contract{
		Name:        ContractName,
		Version:     ContractVersion,
		RuntimeType: commonPb.RuntimeType_WASMER,
	}

	logger := logger2.GetLogger("unit_test")

	return wasmBytes, contractId, logger
}

func fillingBaseParams(parameters map[string][]byte) {
	parameters[protocol.ContractTxIdParam] = []byte("TX_ID")
	parameters[protocol.ContractCreatorOrgIdParam] = []byte("CREATOR_ORG_ID")
	parameters[protocol.ContractCreatorRoleParam] = []byte("CREATOR_ROLE")
	parameters[protocol.ContractCreatorPkParam] = []byte("CREATOR_PK")
	parameters[protocol.ContractSenderOrgIdParam] = []byte("SENDER_ORG_ID")
	parameters[protocol.ContractSenderRoleParam] = []byte("SENDER_ROLE")
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK")
	parameters[protocol.ContractBlockHeightParam] = []byte("111")
}

type SnapshotMock struct {
	cache map[string][]byte
}

func (s SnapshotMock) GetSpecialTxTable() []*commonPb.Transaction {
	panic("implement me")
}

func (s SnapshotMock) GetBlockTimestamp() int64 {
	panic("implement me")
}

func (s SnapshotMock) ApplyTxSimContext(context protocol.TxSimContext, txType protocol.ExecOrderTxType,
	b bool, b2 bool) (bool, int) {
	panic("implement me")
}

func (s SnapshotMock) GetBlockchainStore() protocol.BlockchainStore {
	panic("implement me")
}

func (s SnapshotMock) GetKey(txExecSeq int, contractName string, key []byte) ([]byte, error) {
	combinedKey := fmt.Sprintf("%d-%s-%s", txExecSeq, contractName, key)
	return s.cache[combinedKey], nil
}

func (s SnapshotMock) GetTxRWSetTable() []*commonPb.TxRWSet {
	panic("implement me")
}

func (s SnapshotMock) GetTxResultMap() map[string]*commonPb.Result {
	panic("implement me")
}

func (s SnapshotMock) GetSnapshotSize() int {
	return 0
}

func (s SnapshotMock) GetTxTable() []*commonPb.Transaction {
	panic("implement me")
}

func (s SnapshotMock) GetPreSnapshot() protocol.Snapshot {
	panic("implement me")
}

func (s SnapshotMock) SetPreSnapshot(snapshot protocol.Snapshot) {
	panic("implement me")
}

func (s SnapshotMock) GetBlockHeight() uint64 {
	panic("implement me")
}

func (s SnapshotMock) GetBlockProposer() *accessPb.Member {
	panic("implement me")
}

func (s SnapshotMock) BuildDAG(isSql bool) *commonPb.DAG {
	panic("implement me")
}

func (s SnapshotMock) IsSealed() bool {
	panic("implement me")
}

func (s SnapshotMock) Seal() {
	panic("implement me")
}

func prepareTxSimContext(
	chainId string,
	blockVersion uint32,
	contractName string,
	method string,
	parameters map[string][]byte,
	snapshot protocol.Snapshot) protocol.TxSimContext {

	// 构建 instanceMgrMap
	instanceMgrMap := make(map[commonPb.RuntimeType]protocol.VmInstancesManager)
	wasmerVmPoolManager := NewInstancesManager(chainId)
	instanceMgrMap[commonPb.RuntimeType_WASMER] = wasmerVmPoolManager

	// 构建 chainConfig
	chainConfig := chainconf.ChainConf{
		ChainConf: &config.ChainConfig{
			Contract: &config.ContractConfig{
				EnableSqlSupport: true,
			},
		},
	}

	// 构建 VmManager
	vmManager := vm.NewVmManager(
		instanceMgrMap,
		"",
		nil,
		nil,
		&chainConfig)

	// 处理参数
	params := make([]*commonPb.KeyValuePair, 0)
	for key, val := range parameters {
		params = append(params, &commonPb.KeyValuePair{
			Key:   key,
			Value: val})
	}

	// sender
	sender := accessPb.Member{}

	// 构建 Transaction
	tx := commonPb.Transaction{
		Payload: &commonPb.Payload{
			ChainId:        chainId,
			TxType:         commonPb.TxType_INVOKE_CONTRACT,
			TxId:           uuid.GetUUID(),
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   contractName,
			Method:         method,
			Parameters:     params,
			Sequence:       0,
			Limit:          nil,
		},
		Sender: &commonPb.EndorsementEntry{
			Signer: &sender,
		},
		Result: nil,
	}

	return vm.NewTxSimContext(vmManager, snapshot, &tx, blockVersion, logger2.GetLogger("test"))
}
