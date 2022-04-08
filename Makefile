VERSION=v2.2.0

gomod:
	go get chainmaker.org/chainmaker/chainconf/v2@$(VERSION)
	go get chainmaker.org/chainmaker/common/v2@$(VERSION)
	go get chainmaker.org/chainmaker/localconf/v2@$(VERSION)
	go get chainmaker.org/chainmaker/logger/v2@$(VERSION)
	go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
	go get chainmaker.org/chainmaker/protocol/v2@$(VERSION)
	go get chainmaker.org/chainmaker/store/v2@$(VERSION)
	go get chainmaker.org/chainmaker/utils/v2@$(VERSION)
	go get chainmaker.org/chainmaker/vm/v2@$(VERSION)
	go mod tidy
	cat go.mod|grep chainmaker
