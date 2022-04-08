module chainmaker.org/chainmaker/vm-wasmer-test/v2

go 1.15

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.0
	chainmaker.org/chainmaker/common/v2 v2.2.0
	chainmaker.org/chainmaker/localconf/v2 v2.2.0
	chainmaker.org/chainmaker/logger/v2 v2.2.0
	chainmaker.org/chainmaker/pb-go/v2 v2.2.0
	chainmaker.org/chainmaker/protocol/v2 v2.2.0
	chainmaker.org/chainmaker/store/v2 v2.2.0
	chainmaker.org/chainmaker/utils/v2 v2.2.0
	github.com/Ning-Qing/vm-wasmer/v2 v2.1.1-0.20211213113148-e0a4ba64d0ed
	chainmaker.org/chainmaker/vm/v2 v2.2.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/mitchellh/mapstructure v1.4.2
	github.com/mr-tron/base58 v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/Ning-Qing/vm-wasmer/v2 => ../
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
