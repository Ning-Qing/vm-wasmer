/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tests

import (
	"path/filepath"

	"chainmaker.org/chainmaker/chainconf/v2"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-wasmer-test/v2/accesscontrol"
	"chainmaker.org/chainmaker/vm/v2"
	wasmer "github.com/Ning-Qing/vm-wasmer"
)

func InitVmManager(store protocol.BlockchainStore, chainConfig *chainconf.ChainConf) (protocol.VmManager, error) {
	var err error

	// 初始化 Access Control
	nodeConfig := localconf.ChainMakerConfig.NodeConfig
	skFile := nodeConfig.PrivKeyFile
	if !filepath.IsAbs(skFile) {
		if skFile, err = filepath.Abs(skFile); err != nil {
			return nil, err
		}
	}
	certFile := nodeConfig.CertFile
	if !filepath.IsAbs(certFile) {
		if certFile, err = filepath.Abs(certFile); err != nil {
			return nil, err
		}
	}

	chainId := chainConfig.ChainConfig().ChainId

	acLog := logger.GetLoggerByChain(logger.MODULE_ACCESS, chainId)
	ac, err := accesscontrol.NewAccessControlWithChainConfig(
		chainConfig,
		nodeConfig.OrgId,
		store,
		acLog)
	if err != nil {
		return nil, err
	}

	wasmerVmPoolManager := wasmer.NewInstancesManager(chainId)

	instanceMgrMap := make(map[common.RuntimeType]protocol.VmInstancesManager)
	instanceMgrMap[common.RuntimeType_WASMER] = wasmerVmPoolManager

	vmManager := vm.NewVmManager(
		instanceMgrMap,
		"",
		ac,
		nil,
		chainConfig)

	return vmManager, nil
}
