TEST_TIMEOUT:="70s"
TEST_ARGS ?= -args -logtostderr -v=3
TEST_PKG?=./grpc/...

VERSION=$(shell cat ./VERSION)
LEDGER_NAME:=FINDY_FILE_LEDGER

AUTH_BRANCH=$(shell scripts/branch.sh ../findy-agent-auth/)
GRPC_BRANCH=$(shell scripts/branch.sh ../findy-common-go/)
WRAP_BRANCH=$(shell scripts/branch.sh ../findy-wrapper-go/)

CURRENT_BRANCH=$(shell scripts/branch.sh .)

GO := go
# GO := go1.18beta2
GOBUILD_ARGS:=

COV_FILE:=coverage.txt

SCAN_SCRIPT_URL="https://raw.githubusercontent.com/findy-network/setup-go-action/master/scanner/cp_scan.sh"

scan:
	@curl -s $(SCAN_SCRIPT_URL) | bash

scan_and_report:
	@curl -s $(SCAN_SCRIPT_URL) | bash -s v > licenses.txt

drop_wrap:
	$(GO) mod edit -dropreplace github.com/findy-network/findy-wrapper-go

drop_comm:
	$(GO) mod edit -dropreplace github.com/findy-network/findy-common-go

drop_auth:
	$(GO) mod edit -dropreplace github.com/findy-network/findy-agent-auth

drop_all: drop_auth drop_comm drop_wrap

repl_wrap:
	$(GO) mod edit -replace github.com/findy-network/findy-wrapper-go=../findy-wrapper-go

repl_comm:
	$(GO) mod edit -replace github.com/findy-network/findy-common-go=../findy-common-go

repl_auth:
	$(GO) mod edit -replace github.com/findy-network/findy-agent-auth=../findy-agent-auth

repl_all: repl_auth repl_comm repl_wrap

modules: modules_comm modules_wrap modules_auth

modules_comm: drop_comm
	@echo Syncing modules: findy-common-go/$(GRPC_BRANCH)
	$(GO) get github.com/findy-network/findy-common-go@$(GRPC_BRANCH)

modules_wrap: drop_wrap
	@echo Syncing modules: findy-wrapper-go/$(WRAP_BRANCH)
	$(GO) get github.com/findy-network/findy-wrapper-go@$(WRAP_BRANCH)

modules_auth: drop_auth
	@echo Syncing modules: findy-agent-auth/$(AUTH_BRANCH)
	$(GO) get github.com/findy-network/findy-agent-auth@$(AUTH_BRANCH)

deps:
	$(GO) get -t ./...

update-deps:
	$(GO) get -u ./...

cli:
	@echo "building new CLI by name: fa"
	$(eval VERSION = $(shell cat ./VERSION) $(shell date))
	@echo "Installing version $(VERSION)"
	@$(GO) build \
		-ldflags "-X 'github.com/findy-network/findy-agent/agent/utils.Version=$(VERSION)'" \
		-o $(GOPATH)/bin/fa 

build:
	$(GO) build -v ./...

vet:
	$(GO) vet ./...

shadow:
	@echo Running govet
	$(GO) vet -vettool=$(GOPATH)/bin/shadow ./...
	@echo Govet success

check_fmt:
	$(eval GOFILES = $(shell find . -name '*.go'))
	@gofmt -s -l $(GOFILES)

lint_e:
	@$(GOPATH)/bin/golint ./... | grep -v export | cat

lint:
	@golangci-lint run

test:
	$(GO) test -p 1 -failfast ./...

testr:
	$(GO) test -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./... | tee ../testr.log

test_pkgv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast $(TEST_PKG) $(TEST_ARGS) | tee ../testr.log

test_grpcv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc:
	$(GO) test -timeout $(TEST_TIMEOUT) -p 1 -failfast ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc_rv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc_r:
	$(GO) test -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc_cov_out:
	$(GO) test -p 1 -failfast -timeout $(TEST_TIMEOUT) \
		-coverpkg=github.com/findy-network/findy-agent/... \
		-coverprofile=$(COV_FILE)  \
		-covermode=atomic \
		./grpc/...

test_grpcv_cov_out:
	$(GO) test -v -p 1 -failfast -timeout $(TEST_TIMEOUT) \
		-coverpkg=github.com/findy-network/findy-agent/... \
		-coverprofile=$(COV_FILE)  \
		-covermode=atomic \
		./grpc/... \
		$(TEST_ARGS) | tee ../testr.log

testrv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./... $(TEST_ARGS) | tee ../testr.log

testv:
	$(GO) test -v -p 1 -failfast ./...

test_cov_out:
	$(GO) test -p 1 -failfast -timeout=1200s \
		-coverpkg=github.com/findy-network/findy-agent/... \
		-coverprofile=$(COV_FILE)  \
		-covermode=atomic \
		./...

test_cov: test_cov_out
	$(GO) tool cover -html=$(COV_FILE) -o ./report.html
	open ./report.html


misspell:
	@$(GO) get github.com/client9/misspell 
	@find . -name '*.md' -o -name '*.go' -o -name '*.puml' | xargs misspell -error

e2e: install
	./scripts/e2e/e2e-test.sh init_ledger
	./scripts/e2e/e2e-test.sh e2e
	./scripts/e2e/e2e-test.sh clean

e2e_ci: install
	./scripts/e2e/e2e-test.sh e2e

check: check_fmt vet shadow

install:
	$(eval VERSION = $(shell cat ./VERSION) $(shell date))
	@echo "Installing version $(VERSION)"
	$(GO) install \
		${GOBUILD_ARGS} \
		-ldflags "-X 'github.com/findy-network/findy-agent/agent/utils.Version=$(VERSION)'" \
		./...

image:
	$(eval VERSION = $(shell cat ./VERSION))
	docker build \
		-t findy-agent \
		-f scripts/deploy/Dockerfile .
	docker tag findy-agent:latest findy-agent:$(VERSION)

# **** scripts for local agency development:
# WARNING: this will erase all your local indy wallets
scratch:
	./scripts/dev/dev.sh scratch $(LEDGER_NAME)

run:
	./scripts/dev/dev.sh install_run $(LEDGER_NAME)
# ****

iop:
	gh workflow run iop.yml --ref $(CURRENT_BRANCH)

release:
	gh workflow run do-release.yml
