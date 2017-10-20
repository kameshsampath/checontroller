GO := GO15VENDOREXPERIMENT=1 go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ORG := github.com/kameshsampath
REPOPATH ?= $(ORG)/checontroller
ROOT_PACKAGE = $(GOPATH)/$(REPOPATH)
BUILD_DIR ?= ./bin
GLIDE = glide

glide.lock: 
	glide.yaml | $(ROOT_PACKAGE)
	cd $(ROOT_PACKAGE) && $(GLIDE) update -v 
	@touch $@

vendor: 
	glide.lock | $(ROOT_PACKAGE)
	cd $(ROOT_PACKAGE) && $(GLIDE) --quiet install
	@touch $@

bin/checontroller: bin/checontroller-$(GOOS)-$(GOARCH) 
	cp $(BUILD_DIR)/checontroller-darwin-amd64 $(BUILD_DIR)/checontroller

bin/checontroller-darwin-amd64: gopath 
	GOARCH=amd64 GOOS=darwin go build -o $(BUILD_DIR)/checontroller-darwin-amd64 main.go
	
bin/checontroller-linux-amd64: gopath 
	GOARCH=amd64 GOOS=linux go build -o $(BUILD_DIR)/checontroller-linux-amd64 main.go

bin/checontroller-windows-amd64: gopath 
	GOARCH=amd64 GOOS=windows go build -o $(BUILD_DIR)/checontroller-windows-amd64.exe main.go

.PHONY: gopath
gopath:	$(GOPATH)/src/$(ORG)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: build
build: bin/checontroller

.PHONY: allPF
allPF: bin/checontroller bin/checontroller-darwin-amd64 bin/checontroller-linux-amd64 bin/checontroller-windows-amd64

.PHONY: docker
docker:	bin/checontroller-linux-amd64
	docker build --rm -t "kameshsampath/checontroller:dev" .