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

func TestAsset2(t *testing.T) {

	contractName := "asset"
	contractVersion := "v1.0.0"
	wasmFilePath := "../../testdata/rust-asset-2.0.0.wasm"

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

	testAssetContract(contract, vmManager, snapshotMock, wasmByteCode)
}

func testAssetContract(pbContract *commonPb.Contract,
	vmManager protocol.VmManager,
	snapshotMock *tests.SnapshotMock,
	wasmByteCode []byte) {

	blockchainStoreMock := snapshotMock.GetBlockchainStoreMock()

	// 读取用户证书
	userCertBytes, err := ioutil.ReadFile(UserCertFilePath)
	if err != nil {
		fmt.Printf("#### read '%v' error: %v \n", UserCertFilePath, err)
	}

	//init contract
	fmt.Println("> > > > > > > >  call Contract ==> init_contract()  > > > > > > > >")
	parameters := make(map[string][]byte)
	parameters["issue_limit"] = []byte("200")
	parameters["total_supply"] = []byte("20000")
	parameters["balance"] = []byte("10000")
	parameters["manager_pk"] = []byte(".MANAGER_PK")

	tests.InvokeContractWithParameters("init_contract", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//tests get
	fmt.Println("> > > > > > > >  call Contract ==> get_version()  > > > > > > > >")
	tests.InvokeContractWithParameters("get_version", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> name()  > > > > > > > >")
	tests.InvokeContractWithParameters("name", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> symbol()  > > > > > > > >")
	tests.InvokeContractWithParameters("symbol", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//register 开户
	fmt.Println("> > > > > > > >  call Contract ==> register('SENDER_PK1')  > > > > > > > >")
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK1")
	tests.InvokeContractWithParameters("register", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> register('SENDER_PK2')  > > > > > > > >")
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK2")
	tests.InvokeContractWithParameters("register", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//issue_amount发钱
	fmt.Println("> > > > > > > >  call Contract ==> issue_amount()  'SENDER_PK' ==> 'SENDER_PK1' > > > > > > > >")
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK")
	parameters["to"] = []byte("SENDER_PK1")
	parameters["amount"] = []byte("123")
	tests.InvokeContractWithParameters("issue_amount", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> issue_amount()  'SENDER_PK' ==> 'SENDER_PK1' > > > > > > > >")
	parameters["to"] = []byte("SENDER_PK")
	parameters["amount"] = []byte("160")
	tests.InvokeContractWithParameters("issue_amount", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//transfer 转账
	fmt.Println("> > > > > > > >  call Contract ==> transfer()  'SENDER_PK' ==> 'SENDER_PK1' > > > > > > > >")
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK")
	parameters["to"] = []byte("SENDER_PK1")
	parameters["amount"] = []byte("150")
	tests.InvokeContractWithParameters("transfer", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//issued_amount 当前合约已发金额
	tests.InvokeContractWithParameters("issued_amount", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//balance_of 获取自己账户余额
	parameters["owner"] = []byte("SENDER_PK1")
	tests.InvokeContractWithParameters("balance_of", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()
	//approve 授权spender一定数额
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK1")
	parameters["spender"] = []byte("SENDER_PK2")
	parameters["amount"] = []byte("273")
	tests.InvokeContractWithParameters("approve", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//transfer_from代转账
	parameters[protocol.ContractSenderPkParam] = []byte("SENDER_PK2")
	parameters["from"] = []byte("SENDER_PK1")
	parameters["to"] = []byte("SENDER_PK")
	parameters["amount"] = []byte("272")

	tests.InvokeContractWithParameters("transfer_from", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes)
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()

	//balance_of 获取自己账户余额
	parameters["owner"] = []byte("SENDER_PK1")
	tests.InvokeContractWithParameters("balance_of", pbContract, wasmByteCode, vmManager, snapshotMock,
		parameters, userCertBytes) //result = 1
	fmt.Println("--------------  after ApplyTxSimContext()  ---------------------")
	blockchainStoreMock.PrintDebugInfo()
	fmt.Println()
}
