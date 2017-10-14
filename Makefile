GO := GO15VENDOREXPERIMENT=1 go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ORIGINAL_GOPATH := $(GOPATH)
ORG := github.com/kameshsampath
REPOPATH ?= $(ORG)/che-stack-refresher
ROOT_PACKAGE = $(GOPATH)/$(REPOPATH)
GOPATH  = $(shell pwd)/.gopath
BUILD_DIR ?= ./bin

$(GOPATH)/src/$(ORG):
	mkdir -p $(GOPATH)/src/$(ORG)
	ln -s -f $(shell pwd) $(GOPATH)/src/$(ORG)

bin/che-stack-refresher: bin/che-stack-refresher-$(GOOS)-$(GOARCH) 
	cp $(BUILD_DIR)/che-stack-refresher-darwin-amd64 $(BUILD_DIR)/che-stack-refresher

bin/che-stack-refresher-darwin-amd64: gopath 
	GOARCH=amd64 GOOS=darwin go build -o $(BUILD_DIR)/che-stack-refresher-darwin-amd64 main.go
	
bin/che-stack-refresher-linux-amd64: gopath 
	GOARCH=amd64 GOOS=linux go build -o $(BUILD_DIR)/che-stack-refresher-linux-amd64 main.go

bin/che-stack-refresher-windows-amd64: gopath 
	GOARCH=amd64 GOOS=windows go build -o $(BUILD_DIR)/che-stack-refresher-windows-amd64.exe main.go

.PHONY: gopath
gopath:	$(GOPATH)/src/$(ORG)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: build
build: bin/che-stack-refresher

.PHONY: allPF
allPF: bin/che-stack-refresher bin/che-stack-refresher-darwin-amd64 bin/che-stack-refresher-linux-amd64 bin/che-stack-refresher-windows-amd64

.PHONY: docker
docker:	bin/che-stack-refresher-linux-amd64
	docker build --rm -t "kameshsampath/che-stack-refresher:dev" .