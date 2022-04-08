/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package functional_verify

import (
	"fmt"
	"io/ioutil"
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	tests "chainmaker.org/chainmaker/vm-wasmer-test/v2"
)

/*
	所有交易采用如下默认值：
		ChainId = "chain1"
		OrgId   = "wx-org1.chainmaker.org"

	发送方证书：
		UserCertFilePath = "../../testdata/config/crypto-_config/wx-org1.chainmaker.org/user/client1/client1.sign.crt"
*/
func TestFuncVerify2(t *testing.T) {

	contractName := "func-verify"
	contractVersion := "v1.0.0"
	wasmFilePath := "../../testdata/rust-func-verify-2.1.0.wasm"

	contract, vmManager, snapshot, wasmByteCode, err := tests.BeforeCmdExecute(tests.ChainId, contractName,
		contractVersion, wasmFilePath, ConfigFilePath, UserCertFilePath)
	if err != nil {
		t.Fatal(err)
	}
	blockchainStoreMock := snapshot.GetBlockchainStoreMock()

	// 放入调用合约的数据
	err = tests.PutContractIdIntoStore(blockchainStoreMock, contractName, contract)
	if err != nil {
		t.Fatal(err)
	}
	tests.PutContractBytecodeIntoStore(blockchainStoreMock, contractName, wasmByteCode)
	// 放入被调用合约的数据
	calleeContract := commonPb.Contract{
		Name:        "contract01",
		Version:     "v1.0.0",
		RuntimeType: commonPb.RuntimeType_WASMER,
		Creator:     contract.Creator,
	}
	err = tests.PutContractIdIntoStore(blockchainStoreMock, "contract01", &calleeContract)
	if err != nil {
		t.Fatal(err)
	}
	// 跨合约调用测试中写死了合约名称为'contract01'
	tests.PutContractBytecodeIntoStore(blockchainStoreMock, "contract01", wasmByteCode)

	testFuncVerifyContract(contract, vmManager, snapshot, wasmByteCode)
}

func testFuncVerifyContract(pbContract *commonPb.Contract,
	vmManager protocol.VmManager,
	snapshotMock *tests.SnapshotMock,
	wasmByteCode []byte) {

	blockchainStore := snapshotMock.GetBlockchainStoreMock()

	// 读取用户证书
	userCertBytes, err := ioutil.ReadFile(UserCertFilePath)
	if err != nil {
		fmt.Printf("#### read '%v' error: %v \n", UserCertFilePath, err)
	}

	//init pbContract
	fmt.Println("> > > > > > > >  call Contract ==> init_context()  > > > > > > > >")
	args := make(map[string][]byte)

	tests.InvokeContractWithParameters("init_contract", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	//test_put_state
	fmt.Println("> > > > > > > >  call Contract ==> test_put_state()  > > > > > > > >")
	args["field"] = []byte("field1234")
	args["value"] = []byte("value100")

	tests.InvokeContractWithParameters("test_put_state", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> test_put_state()  > > > > > > > >")
	args["key"] = []byte("key1235")
	args["value"] = []byte("value101")

	tests.InvokeContractWithParameters("test_put_state", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	//test_put_pre_state
	fmt.Println("> > > > > > > >  call Contract ==> test_put_pre_state()  > > > > > > > >")
	args["key"] = []byte("key")
	args["field"] = []byte("aabcd1234")
	args["value"] = []byte("value1")

	tests.InvokeContractWithParameters("test_put_pre_state", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> test_put_pre_state()  > > > > > > > >")
	args["field"] = []byte("aabcd1235")
	args["value"] = []byte("value2")

	tests.InvokeContractWithParameters("test_put_pre_state", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	fmt.Println("> > > > > > > >  call Contract ==> test_kv_iterator()  > > > > > > > >")
	//test_kv_iterator 测试kv迭代器
	args["key"] = []byte("aabcd1234")
	args["limit"] = []byte("aabcd1235")
	//tests.InvokeContractWithParameters("test_kv_iterator", pbContract, wasmByteCode, vmManager, snapshot,
	//	args, tests.UserCertFilePath)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	//test_iter_pre_field
	fmt.Println("> > > > > > > >  call Contract ==> test_iter_pre_field()  > > > > > > > >")
	//InvokeContractWithParameters("test_iter_pre_field", pbContract, wasmByteCode, vmManager, snapshot,
	//args, tests.UserCertFilePath)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	//test_iter_pre_key
	fmt.Println("> > > > > > > >  call Contract ==> test_iter_pre_key()  > > > > > > > >")
	//InvokeContractWithParameters("test_iter_pre_key", pbContract, wasmByteCode, vmManager, snapshot,
	//args, tests.UserCertFilePath)
	blockchainStore.PrintDebugInfo()
	fmt.Println()

	//functional_verify
	fmt.Println("> > > > > > > >  call Contract ==> functional_verify()  > > > > > > > >")
	args["fileHash"] = []byte("file_hash")

	tests.InvokeContractWithParameters("functional_verify", pbContract, wasmByteCode, vmManager, snapshotMock,
		args, userCertBytes)
	blockchainStore.PrintDebugInfo()
	fmt.Println()
}
