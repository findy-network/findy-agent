IMAGE_NAME?=findy-agent
VERSION=$(shell cat ./VERSION)
FINDY_WRAPPER_GO_VERSION?=f011c1caaca

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

image:
	-git clone git@github.com:findy-network/findy-wrapper-go.git .docker/findy-wrapper-go
	cd .docker/findy-wrapper-go && git -c advice.detachedHead=false checkout $(FINDY_WRAPPER_GO_VERSION)
	docker build -t $(IMAGE_NAME) .


# **** scripts for local development:
# WARNING: this will erase all your local indy wallets
scratch:
	./tools/dev.sh scratch

run:
	./tools/dev.sh install_run
# ****
