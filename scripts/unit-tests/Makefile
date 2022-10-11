VON_NETWORK_VERSION:="22973513c99cc9a286a6f181ca5c5f354a4eb2ee"

test: von-up build test-only

test-only:
	@echo "Start test in docker container"
	docker run -it --rm -v $(PWD)/../../:/work findy-agent-unit-tests 

build:
	cd ../.. && \
		docker build -t findy-agent-unit-tests -f ./scripts/unit-tests/Dockerfile .

clone:
	mkdir -p .docker
	-git clone https://github.com/bcgov/von-network .docker/von-network
	cd .docker/von-network && git reset --hard $(VON_NETWORK_VERSION)

von-up: clone
# TODO: use von-network image when ARM is supported
#	cd .docker/von-network && ./manage build && ./manage up
	docker pull ghcr.io/lauravuo/von-network:latest
	docker tag ghcr.io/lauravuo/von-network:latest von-network-base
	cd .docker/von-network && ./manage up
	../dev/docker/wait-for-ledger.sh

down:
	cd .docker/von-network && ./manage down
