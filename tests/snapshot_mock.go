/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tests

import (
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
)

type SnapshotMock struct {
	blockchainStore *BlockchainStoreMock
}

func (s *SnapshotMock) GetSpecialTxTable() []*common.Transaction {
	panic("implement me")
}

func (s *SnapshotMock) GetBlockTimestamp() int64 {
	panic("implement me")
}

func (s *SnapshotMock) GetBlockchainStoreMock() *BlockchainStoreMock {
	return s.blockchainStore
}

func (s *SnapshotMock) GetBlockchainStore() protocol.BlockchainStore {
	return s.blockchainStore
}

func (s *SnapshotMock) GetKey(_txExecSeq int, contractName string, key []byte) ([]byte, error) {
	return s.blockchainStore.ReadObject(contractName, key)
}

func (s *SnapshotMock) GetTxRWSetTable() []*common.TxRWSet {
	panic("implement me")
}

func (s *SnapshotMock) GetTxResultMap() map[string]*common.Result {
	panic("implement me")
}

func (s *SnapshotMock) GetSnapshotSize() int {
	return 0
}

func (s *SnapshotMock) GetTxTable() []*common.Transaction {
	panic("implement me")
}

func (s *SnapshotMock) GetPreSnapshot() protocol.Snapshot {
	panic("implement me")
}

func (s *SnapshotMock) SetPreSnapshot(snapshot protocol.Snapshot) {
	panic("implement me")
}

func (s *SnapshotMock) GetBlockHeight() uint64 {
	return 0
}

func (s *SnapshotMock) GetBlockProposer() *pbac.Member {
	panic("implement me")
}

func (s *SnapshotMock) ApplyTxSimContext(txSimContext protocol.TxSimContext, txType protocol.ExecOrderTxType,
	b bool, txSuccess bool) (bool, int) {
	s.blockchainStore.Lock()
	defer s.blockchainStore.Unlock()

	txRWSet := txSimContext.GetTxRWSet(txSuccess)
	for _, txWrite := range txRWSet.TxWrites {
		combinedKey := constructKey(txWrite.ContractName, txWrite.Key)
		s.blockchainStore.Cache[combinedKey] = txWrite.Value
	}

	return true, 0
}

func (s *SnapshotMock) BuildDAG(isSql bool) *common.DAG {
	panic("implement me")
}

func (s *SnapshotMock) IsSealed() bool {
	panic("implement me")
}

func (s *SnapshotMock) Seal() {
	panic("implement me")
}

func loadSnapshot() (*SnapshotMock, error) {
	return &SnapshotMock{
		blockchainStore: &BlockchainStoreMock{
			Cache:      make(map[string][]byte),
			readCount:  0,
			writeCount: 0,
		},
	}, nil
}
