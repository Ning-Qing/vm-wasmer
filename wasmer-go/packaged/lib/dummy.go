// See https://github.com/golang/go/issues/26366.
package lib

import (
	_ "github.com/Ning-Qing/vm-wasmer/wasmer-go/packaged/lib/darwin-amd64"
	_ "github.com/Ning-Qing/vm-wasmer/wasmer-go/packaged/lib/linux-aarch64"
	_ "github.com/Ning-Qing/vm-wasmer/wasmer-go/packaged/lib/linux-amd64"
	_ "github.com/Ning-Qing/vm-wasmer/wasmer-go/packaged/lib/linux-musl-amd64"
)
