VERSION=$(shell cat ./VERSION)

API_BRANCH=$(shell ./branch.sh ../findy-agent-api/)
GRPC_BRANCH=$(shell ./branch.sh ../findy-grpc/)

modules:
	@echo Syncing modules for work brances ...
	go get github.com/findy-network/findy-agent-api@$(API_BRANCH)
	go get github.com/findy-network/findy-grpc@$(GRPC_BRANCH)

deps:
	go get -t ./...

update-deps:
	go get -u ./...

build:
	go build -v ./...

vet:
	go vet ./...

install:
	@echo "Installing version $(VERSION)"
	go install \
		-ldflags "-X 'github.com/findy-network/findy-agent/agent/utils.Version=$(VERSION)'" \
		./...

shadow:
	@echo Running govet
	go vet -vettool=$(GOPATH)/bin/shadow ./...
	@echo Govet success

check_fmt:
	$(eval GOFILES = $(shell find . -name '*.go'))
	@gofmt -l $(GOFILES)

lint:
	$(GOPATH)/bin/golint ./... 

lint_e:
	@$(GOPATH)/bin/golint ./... | grep -v export | cat

test:
	go test -v -p 1 -failfast ./...

test_cov:
	go test -v -p 1 -failfast -coverprofile=c.out ./... && go tool cover -html=c.out

check: check_fmt vet shadow

