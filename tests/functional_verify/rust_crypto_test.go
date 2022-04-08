/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package functional_verify

import (
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	tests "chainmaker.org/chainmaker/vm-wasmer-test/v2"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestCrypto2(t *testing.T) {

	contractName := "asset"
	contractVersion := "v1.0.0"
	wasmFilePath := "../../testdata/rust-crypto-2.0.0.wasm"

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

	testCryptoContract(contract, vmManager, snapshotMock, wasmByteCode)
}

func testCryptoContract(pbContract *commonPb.Contract,
	vmManager protocol.VmManager,
	snapshotMock *tests.SnapshotMock,
	wasmByteCode []byte) {

	blockchainStoreMock := snapshotMock.GetBlockchainStoreMock()

	// 读取用户证书
	userCertBytes, err := ioutil.ReadFile(UserCertFilePath)
	if err != nil {
		fmt.Printf("#### read '%v' error: %v \n", UserCertFilePath, err)
	}

	parameters := make(map[string][]byte)

	//init contract
	fmt.Println("> > > > > > > >  call Contract ==> init_contract()  > > > > > > > >")
	tests.InvokeContractWithParameters("init_contract", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//tests get
	parameters["value"] = []byte("567124123")
	parameters["sign"] = []byte("MEUCIEGi5PH4Sum9v4AL5ob+lq4jiwRseWtYi4gEtjnSb0BFAiEAip7z7UJE/clX9gX2ndNJopSVDNyyRKfIeoi1LQte7aM=")
	parameters["originalData"] = []byte("052cce07a3c544558a29f4d6062b4f00")
	parameters["publicKeyXy"] = []byte("BEm4d5Cdy3oF79O6gwLI/n3N0jClaKnHUHKzWo8Gas4Y/J8wBiOPP92Uii/rumYt5my+xKZuCRlgjJ+o8W0CUAE=")
	parameters["forever"] = []byte("true")
	parameters["contract_name"] = []byte("taifu_contract")
	fmt.Println("> > > > > > > >  call Contract ==> save_auth_data()  > > > > > > > >")
	tests.InvokeContractWithParameters("save_auth_data", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()
}
