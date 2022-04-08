/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	accessPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/store/v2"
	"chainmaker.org/chainmaker/store/v2/conf"
	"chainmaker.org/chainmaker/utils/v2"
	"chainmaker.org/chainmaker/vm/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
)

const (
	OrgId        = "wx-org1.chainmaker.org"
	ChainId      = "chain1"
	BlockVersion = uint32(1)
)

func PrepareMember(orgId string, certBytes []byte) (*accessPb.Member, error) {
	// 构建 Sender
	sender := &accessPb.Member{
		OrgId:      orgId,
		MemberType: accessPb.MemberType_CERT,
		MemberInfo: certBytes,
	}

	return sender, nil
}

func PrepareTx(sender *accessPb.Member, chainId string,
	contractName string, method string, args map[string][]byte,
	seq uint64, limit *commonPb.Limit) *commonPb.Transaction {

	txId := uuid.GetUUID()

	params := make([]*commonPb.KeyValuePair, 0)
	for key, val := range args {
		params = append(params, &commonPb.KeyValuePair{
			key, val,
		})
	}

	return &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ChainId:        chainId,
			TxType:         commonPb.TxType_INVOKE_CONTRACT,
			TxId:           txId,
			Timestamp:      0,
			ExpirationTime: 0,
			ContractName:   contractName,
			Method:         method,
			Parameters:     params,
			Sequence:       seq,
			Limit:          limit,
		},
		Sender: &commonPb.EndorsementEntry{
			Signer: sender,
		},
		Result: nil,
	}
}

func BeforeCmdExecute(chainId, contractName, contractVersion, wasmFilePath, configFilePath, userCertFilePath string) (
	*commonPb.Contract, protocol.VmManager, *SnapshotMock, []byte, error) {
	// 加载 Config 文件
	if err := initLocalConfig(configFilePath); err != nil {
		panic(fmt.Sprintf("initLocalConfig() error: %v.", err))
	}
	// 利用 StorageConfig 配置项
	// 初始化 blockchainStore 系统
	blockchainStore, err := initStore(chainId)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer blockchainStore.Close()

	// 初始化 ChainConfig 对象
	chainConf, err := chainconf.Genesis(localconf.ChainMakerConfig.BlockChainConfig[0].Genesis)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	genesis, rwset, err := utils.CreateGenesis(chainConf)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	err = blockchainStore.InitGenesis(&storePb.BlockWithRWSet{Block: genesis, TxRWSets: rwset,
		ContractEvents: []*commonPb.ContractEvent{}})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	chainConfig, err := initChainConfig(chainId, blockchainStore)
	vmManager, err := InitVmManager(blockchainStore, chainConfig)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	snapshot, err := loadSnapshot()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	log := logger.GetLogger("monitor")
	pbContract, wasmBytecode, err := InitWasmContract(
		contractName,
		contractVersion,
		wasmFilePath,
		localconf.ChainMakerConfig.NodeConfig.OrgId,
		userCertFilePath,
		log)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return pbContract, vmManager, snapshot, wasmBytecode, nil
}

func initLocalConfig(configFilePath string) error {
	cmd := &cobra.Command{}
	localconf.ConfigFilepath = configFilePath
	return localconf.InitLocalConfig(cmd)
}

func initChainConfig(chainId string, store protocol.BlockchainStore) (*chainconf.ChainConf, error) {
	chainConf, err := chainconf.NewChainConf(
		chainconf.WithChainId(chainId),
		chainconf.WithBlockchainStore(store),
	)
	if err != nil {
		return nil, err
	}
	if err = chainConf.Init(); err != nil {
		return nil, err
	}

	return chainConf, nil
}

func initStore(chainId string) (protocol.BlockchainStore, error) {
	storageConfig := conf.StorageConfig{}
	err := mapstructure.Decode(localconf.ChainMakerConfig.StorageConfig, &storageConfig)
	if err != nil {
		return nil, err
	}
	storageConfig.StorePath = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Nanosecond()))
	var storeFactory store.Factory
	storeLogger := logger.GetLoggerByChain(logger.MODULE_STORAGE, chainId)
	blockchainStore, err := storeFactory.NewStore(
		chainId,
		&storageConfig,
		storeLogger,
		nil)
	if err != nil {
		return nil, err
	}

	return blockchainStore, nil
}

func InitWasmContract(
	contractName string, // 合约信息
	contractVersion string, // 合约信息
	wasmFilePath string, // 合约信息
	orgId string, // 操作人信息
	certFilePath string, // 操作人信息
	log *logger.CMLogger) (*commonPb.Contract, []byte, error) {

	runtimeType := commonPb.RuntimeType_WASMER

	wasmFile, err := os.Open(wasmFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("can not read wasm file %v", wasmFilePath)
	}
	wasmByteCode, _ := ioutil.ReadAll(wasmFile)
	log.Debugf("Wasm bytecode size=%d\n", len(wasmByteCode))

	certFile, err := ioutil.ReadFile(certFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("can not read certFilePath %v", certFilePath)
	}

	sender := &accessPb.MemberFull{
		OrgId:      orgId,
		MemberInfo: certFile,
		//IsFullCert: true,
	}

	pbContract := commonPb.Contract{
		Name:        contractName,
		Version:     contractVersion,
		RuntimeType: runtimeType,
		Creator:     sender,
	}

	return &pbContract, wasmByteCode, nil
}

func InvokeContractWithParameters(method string, pbContract *commonPb.Contract, wasmByteCode []byte,
	vmManager protocol.VmManager, snapshot protocol.Snapshot, args map[string][]byte,
	userCertBytes []byte) *commonPb.ContractResult {

	// 构建 sender
	sender, err := PrepareMember(OrgId, userCertBytes)
	if err != nil {
		panic(fmt.Sprintf("InvokeContractWithParameters => PrepareMember error: %v", err))
	}

	// 构建 Tx
	tx := PrepareTx(sender, ChainId, pbContract.Name, method, args, 0, nil)

	// 构建 TxSimContext
	txSimContext := vm.NewTxSimContext(vmManager, snapshot, tx, BlockVersion, logger.GetLogger("test"))

	result, _, _ := vmManager.RunContract(pbContract, method, wasmByteCode, args, txSimContext, 0,
		commonPb.TxType_INVOKE_CONTRACT)
	//fmt.Printf("result = %+v \n", result)
	//fmt.Printf("code = %+v \n", code)
	//fmt.Println("---------------  After Invoke Contract  --------------------")
	//vm.PrintTxReadSet(txSimContext)
	//vm.PrintTxWriteSet(txSimContext)

	//snapshot.ApplyTxSimContext(txSimContext, true)
	return result
}

const contractStoreSeparator = '#'

func constructKey(contractName string, key []byte) string {
	return string(append(append([]byte(contractName), contractStoreSeparator), key...))
}
