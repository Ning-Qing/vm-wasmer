//go:build !custom_wasmer_runtime
// +build !custom_wasmer_runtime

package wasmer

// #cgo CFLAGS: -I${SRCDIR}/packaged/include
// #cgo LDFLAGS: -lwasmer
//
// #cgo linux,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/linux-musl-amd64 -L${SRCDIR}/packaged/lib/linux-musl-amd64
// #cgo linux,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/linux-aarch64 -L${SRCDIR}/packaged/lib/linux-aarch64
// #cgo darwin,amd64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/darwin-amd64 -L${SRCDIR}/packaged/lib/darwin-amd64
// #cgo darwin,arm64 LDFLAGS: -Wl,-rpath,${SRCDIR}/packaged/lib/darwin-aarch64 -L${SRCDIR}/packaged/lib/darwin-aarch64
//
// #include <wasmer.h>
import "C"

// See https://github.com/golang/go/issues/26366.
import (
	_ "github.com/Ning-Qing/vm-wasmer/v2/wasmer-go/packaged/include"
	_ "github.com/Ning-Qing/vm-wasmer/v2/wasmer-go/packaged/lib"
)
