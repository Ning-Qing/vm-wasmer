/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tests

import (
	"fmt"
	"strings"
	"sync"

	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

type BlockchainStoreMock struct {
	mutex      sync.RWMutex
	readCount  int
	writeCount int
	Cache      map[string][]byte
}

func PutContractIdIntoStore(store *BlockchainStoreMock, name string, pbContract *commonPb.Contract) error {
	store.Lock()
	defer store.Unlock()

	bytes, err := pbContract.Marshal()
	if err != nil {
		return err
	}
	combineKey := string(utils.GetContractDbKey(name))
	store.Cache[combineKey] = bytes

	return nil
}

func PutContractBytecodeIntoStore(store *BlockchainStoreMock, name string, bytecode []byte) {
	store.Lock()
	defer store.Unlock()

	combineKey := string(utils.GetContractByteCodeDbKey(name))
	store.Cache[combineKey] = bytecode
}

func (s *BlockchainStoreMock) RLock() {
	s.mutex.RLock()
	s.readCount++
	//fmt.Printf("BlockchainStoreMock => Read Locked => read count(%v), write count(%v) \n", s.readCount, s.writeCount)
}

func (s *BlockchainStoreMock) RUnlock() {
	s.mutex.RUnlock()
	s.readCount--
	//fmt.Printf("BlockchainStoreMock Read Lock UnLocked => read count(%v), write count(%v) \n", s.readCount, s.writeCount)
}

func (s *BlockchainStoreMock) Lock() {
	s.mutex.Lock()
	s.writeCount++
	//fmt.Printf("BlockchainStoreMock Write Lock Locked => read count(%v), write count(%v) \n", s.readCount, s.writeCount)
}

func (s *BlockchainStoreMock) Unlock() {
	s.mutex.Unlock()
	s.writeCount--
	//fmt.Printf("BlockchainStoreMock Write Lock UnLocked => read count(%v), write count(%v) \n", s.readCount, s.writeCount)
}

func (s *BlockchainStoreMock) PrintDebugInfo() {
	s.RLock()
	defer s.RUnlock()

	for key, val := range s.Cache {
		if strings.Index(key, utils.PrefixContractInfo) >= 0 || strings.Index(key, utils.PrefixContractByteCode) >= 0 {
			fmt.Printf("[block store]: %v ==> byte(%d) \n", key, len(val))
		} else {
			fmt.Printf("[block store]: %v ==> %s \n", key, val)
		}
	}
}

func (b *BlockchainStoreMock) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) ExecDdlSql(contractName, sql, version string) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) CommitDbTransaction(txName string) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) RollbackDbTransaction(txName string) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) CreateDatabase(contractName string) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) DropDatabase(contractName string) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetContractDbName(contractName string) string {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetContractByName(name string) (*commonPb.Contract, error) {
	b.RLock()
	defer b.RUnlock()

	combineKey := string(utils.GetContractDbKey(name))
	if bytes, ok := b.Cache[combineKey]; ok {
		pbContract := commonPb.Contract{}
		if err := pbContract.Unmarshal(bytes); err != nil {
			return nil, fmt.Errorf("unmarshal commonOb.Contract error: %v", err)
		}
		return &pbContract, nil
	}
	return nil, fmt.Errorf("can't find contract '%s'", name)
}

func (b *BlockchainStoreMock) GetContractBytecode(name string) ([]byte, error) {
	b.RLock()
	defer b.RUnlock()

	combineKey := string(utils.GetContractByteCodeDbKey(name))
	if bytes, ok := b.Cache[combineKey]; ok {
		return bytes, nil
	}
	return nil, fmt.Errorf("can't find contract bytecode '%s'", name)
}

func (b *BlockchainStoreMock) GetMemberExtraData(member *pbac.Member) (*pbac.MemberExtraData, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) InitGenesis(genesisBlock *store.BlockWithRWSet) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) BlockExists(blockHash []byte) (bool, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetHeightByHash(blockHash []byte) (uint64, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetBlock(height uint64) (*commonPb.Block, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetLastConfigBlock() (*commonPb.Block, error) {
	panic("implement me")
}

func (b *BlockchainStoreMock) GetLastChainConfig() (*config.ChainConfig, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetBlockByTx(txId string) (*commonPb.Block, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetBlockWithRWSets(height uint64) (*store.BlockWithRWSet, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTx(txId string) (*commonPb.Transaction, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) TxExists(txId string) (bool, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxHeight(txId string) (uint64, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxConfirmedTime(txId string) (int64, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetLastBlock() (*commonPb.Block, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) ReadObject(contractName string, key []byte) ([]byte, error) {
	//fmt.Printf("BlockchainStoreMock::ReadObject() was called. \n")
	b.RLock()
	defer b.RUnlock()

	//fmt.Printf("BlockchainStoreMock::ReadObject() was called => contractName = %v, key = %s \n", contractName, key)
	combinedKey := constructKey(contractName, key)
	return b.Cache[combinedKey], nil
}

func (b BlockchainStoreMock) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetDBHandle(dbName string) protocol.DBHandle {
	panic("implement me")
}

func (b BlockchainStoreMock) GetArchivedPivot() uint64 {
	panic("implement me")
}

func (b BlockchainStoreMock) ArchiveBlock(archiveHeight uint64) error {
	panic("implement me")
}

func (b BlockchainStoreMock) RestoreBlocks(serializedBlocks [][]byte) error {
	panic("implement me")
}

func (b BlockchainStoreMock) Close() error {
	panic("implement me")
}

func (b BlockchainStoreMock) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxInfoOnly(txId string) (*commonPb.TransactionInfo, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxInfoWithRWSet(txId string) (*commonPb.TransactionInfoWithRWSet, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxWithRWSet(txId string) (*commonPb.TransactionWithRWSet, error) {
	panic("implement me")
}

func (b BlockchainStoreMock) GetTxWithInfo(txId string) (*commonPb.TransactionInfo, error) {
panic("implement me")
}