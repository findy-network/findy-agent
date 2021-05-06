VERSION=$(shell cat ./VERSION)
LEDGER_NAME:=FINDY_FILE_LEDGER

API_BRANCH=$(shell ./branch.sh ../findy-agent-api/)
GRPC_BRANCH=$(shell ./branch.sh ../findy-common-go/)
WRAP_BRANCH=$(shell ./branch.sh ../findy-wrapper-go/)

scan:
	@./scan.sh

drop_wrap:
	go mod edit -dropreplace github.com/findy-network/findy-wrapper-go

drop_comm:
	go mod edit -dropreplace github.com/findy-network/findy-common-go

drop_api:
	go mod edit -dropreplace github.com/findy-network/findy-agent-api

drop_all: drop_api drop_comm drop_wrap

repl_wrap:
	go mod edit -replace github.com/findy-network/findy-wrapper-go=../findy-wrapper-go

repl_comm:
	go mod edit -replace github.com/findy-network/findy-common-go=../findy-common-go

repl_api:
	go mod edit -replace github.com/findy-network/findy-agent-api=../findy-agent-api

repl_all: repl_api repl_comm repl_wrap

modules: modules_comm modules_wrap modules_api

modules_comm: drop_comm
	@echo Syncing modules: findy-common-api/$(GRPC_BRANCH)
	go get github.com/findy-network/findy-common-go@$(GRPC_BRANCH)

modules_wrap:
	@echo Syncing modules: findy-wrapper-go/$(WRAP_BRANCH)
	go get github.com/findy-network/findy-wrapper-go@$(WRAP_BRANCH)

modules_api: 
	@echo Syncing modules: findy-agent-api/$(API_BRANCH)
	go get github.com/findy-network/findy-agent-api@$(API_BRANCH)

deps:
	go get -t ./...

update-deps:
	go get -u ./...

cli:
	go build -o $(GOPATH)/bin/cli

build:
	go build -v ./...

vet:
	go vet ./...

shadow:
	@echo Running govet
	go vet -vettool=$(GOPATH)/bin/shadow ./...
	@echo Govet success

check_fmt:
	$(eval GOFILES = $(shell find . -name '*.go'))
	@gofmt -s -l $(GOFILES)

lint_e:
	@$(GOPATH)/bin/golint ./... | grep -v export | cat

lint:
	@golangci-lint run

test:
	go test -p 1 -failfast ./...

testv:
	go test -v -p 1 -failfast ./...

test_cov:
	go test -v -p 1 -failfast -coverprofile=c.out ./... && go tool cover -html=c.out

e2e: install
	./scripts/dev/e2e-test.sh init_ledger
	./scripts/dev/e2e-test.sh e2e
	./scripts/dev/e2e-test.sh clean

e2e_ci: install
	./scripts/dev/e2e-test.sh e2e

check: check_fmt vet shadow

install:
	$(eval VERSION = $(shell cat ./VERSION))
	@echo "Installing version $(VERSION)"
	go install \
		-ldflags "-X 'github.com/findy-network/findy-agent/agent/utils.Version=$(VERSION)'" \
		./...

image:
	# https prefix for go build process to be able to clone private modules
	@[ "${HTTPS_PREFIX}" ] || ( echo "ERROR: HTTPS_PREFIX <{githubUser}:{githubToken}@> is not set"; exit 1 )
	$(eval VERSION = $(shell cat ./VERSION))
	docker build --build-arg HTTPS_PREFIX=$(HTTPS_PREFIX) -t findy-agent .
	docker tag findy-agent:latest findy-agent:$(VERSION)

agency: image
	$(eval VERSION = $(shell cat ./VERSION))
	docker build -t findy-agency --build-arg CLI_VERSION=$(VERSION) ./scripts/deploy
	docker tag findy-agency:latest findy-agency:$(VERSION)

run-agency: 
	echo "{}" > findy.json && \
	docker run -it --rm \
		-e FCLI_AGENCY_SALT="this is only example" \
		-p 8080:8080 \
		-p 50052:50051 \
		-v $(PWD)/scripts/dev/genesis_transactions:/genesis_transactions \
		-v $(PWD)/scripts/dev/steward.exported:/steward.exported \
		-v $(PWD)/findy.json:/root/findy.json findy-agency

# **** scripts for local agency development:
# WARNING: this will erase all your local indy wallets
scratch:
	./scripts/dev/dev.sh scratch $(LEDGER_NAME)

run:
	./scripts/dev/dev.sh install_run $(LEDGER_NAME)
# ****
