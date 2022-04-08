// See https://github.com/golang/go/issues/26366.
package lib

import (
	_ "chainmaker.org/chainmaker/vm-wasmer/v2/wasmer-go/packaged/lib/darwin-amd64"
	_ "chainmaker.org/chainmaker/vm-wasmer/v2/wasmer-go/packaged/lib/linux-aarch64"
	_ "chainmaker.org/chainmaker/vm-wasmer/v2/wasmer-go/packaged/lib/linux-amd64"
	_ "chainmaker.org/chainmaker/vm-wasmer/v2/wasmer-go/packaged/lib/linux-musl-amd64"
)
