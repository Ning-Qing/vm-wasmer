VERSION=v2.1.0

gomod:
		go get chainmaker.org/chainmaker/common/v2@$(VERSION)
		go get chainmaker.org/chainmaker/logger/v2@$(VERSION)
		go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
		go get chainmaker.org/chainmaker/protocol/v2@v2.1.0_alpha_fix
		go get chainmaker.org/chainmaker/store/v2@$(VERSION)
		go get chainmaker.org/chainmaker/utils/v2@$(VERSION)
		go get chainmaker.org/chainmaker/vm/v2@v2.1.0_alpha_fix
		go mod tidy
		cat go.mod|grep chainmaker

