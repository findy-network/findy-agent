TEST_TIMEOUT:="70s"
TEST_ARGS ?= -args -logtostderr -v=3

VERSION=$(shell cat ./VERSION)
LEDGER_NAME:=FINDY_FILE_LEDGER

API_BRANCH=$(shell scripts/branch.sh ../findy-agent-api/)
GRPC_BRANCH=$(shell scripts/branch.sh ../findy-common-go/)
WRAP_BRANCH=$(shell scripts/branch.sh ../findy-wrapper-go/)

CURRENT_BRANCH=$(shell scripts/branch.sh .)

GO := go
# GO := go1.18beta2

scan:
	@scripts/scan.sh $(ARGS)

drop_wrap:
	$(GO) mod edit -dropreplace github.com/findy-network/findy-wrapper-go

drop_comm:
	$(GO) mod edit -dropreplace github.com/findy-network/findy-common-go

drop_api:
	$(GO) mod edit -dropreplace github.com/findy-network/findy-agent-api

drop_all: drop_api drop_comm drop_wrap

repl_wrap:
	$(GO) mod edit -replace github.com/findy-network/findy-wrapper-go=../findy-wrapper-go

repl_comm:
	$(GO) mod edit -replace github.com/findy-network/findy-common-go=../findy-common-go

repl_api:
	$(GO) mod edit -replace github.com/findy-network/findy-agent-api=../findy-agent-api

repl_all: repl_api repl_comm repl_wrap

modules: modules_comm modules_wrap modules_api

modules_comm: drop_comm
	@echo Syncing modules: findy-common-go/$(GRPC_BRANCH)
	$(GO) get github.com/findy-network/findy-common-go@$(GRPC_BRANCH)

modules_wrap: drop_wrap
	@echo Syncing modules: findy-wrapper-go/$(WRAP_BRANCH)
	$(GO) get github.com/findy-network/findy-wrapper-go@$(WRAP_BRANCH)

modules_api: drop_api
	@echo Syncing modules: findy-agent-api/$(API_BRANCH)
	$(GO) get github.com/findy-network/findy-agent-api@$(API_BRANCH)

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

test_grpcv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc:
	$(GO) test -timeout $(TEST_TIMEOUT) -p 1 -failfast ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc_rv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./grpc/... $(TEST_ARGS) | tee ../testr.log

test_grpc_r:
	$(GO) test -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./grpc/... $(TEST_ARGS) | tee ../testr.log

testrv:
	$(GO) test -v -timeout $(TEST_TIMEOUT) -p 1 -failfast -race ./... $(TEST_ARGS) | tee ../testr.log

testv:
	$(GO) test -v -p 1 -failfast ./...

test_cov_out:
	$(GO) test -p 1 -failfast \
		-coverpkg=github.com/findy-network/findy-agent/... \
		-coverprofile=coverage.txt  \
		-covermode=atomic \
		./...

test_cov: test_cov_out
	$(GO) tool cover -html=coverage.txt

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
