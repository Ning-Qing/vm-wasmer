/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

// #cgo CFLAGS: -I${SRCDIR}/../../wasmer-go/packaged/include
// #cgo LDFLAGS: -lwasmer
//
// #cgo linux,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/linux-amd64 -L${SRCDIR}/../../wasmer-go/packaged/lib/linux-amd64
// #cgo linux,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/linux-aarch64 -L${SRCDIR}/../../wasmer-go/packaged/lib/linux-aarch64
// #cgo darwin,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/darwin-amd64 -L${SRCDIR}/../../wasmer-go/packaged/lib/darwin-amd64
// #cgo darwin,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/darwin-aarch64 -L${SRCDIR}/../../wasmer-go/packaged/lib/darwin-aarch64
//
// #include <wasmer.h>
import "C"

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/Ning-Qing/vm-wasmer/v2/wasmer-go"
)

func main() {
	values := []wasmer.Value{
		wasmer.NewI64(int64(1)),
		wasmer.NewI64(int64(2)),
		wasmer.NewI64(int64(3)),
	}
	fmt.Sprintf("%v", &values)

	fmt.Printf("wait for start ")
	for i := 0; i < 15; i++ {
		time.Sleep(time.Second)
		fmt.Printf(".")
	}
	fmt.Println()

	for i := 0; i < 1000000; i++ {

		vec := wasmer.ToValueVec(values)
		time.Sleep(100)

		wasmer.DeleteValueVec(vec)
		time.Sleep(100)
	}

	for i := 0; i < 3; i++ {
		time.Sleep(time.Second * 1)
		runtime.GC()
	}
	debug.FreeOSMemory()

	fmt.Printf("wait for end ")
	for i := 0; i < 100; i++ {
		time.Sleep(time.Second)
		fmt.Printf(".")
	}
}
