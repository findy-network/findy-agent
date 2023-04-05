VON_NETWORK_VERSION:=7365a190a72edccccdffcdbce95e043cb8d9ac63

FCLI:="findy-agent-cli"
AUTH_URL:="http://localhost:8888"
AUTH_KEY:="15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c"
AUTH_ORIGIN:="localhost:8888"

gen-yml: docker-host
# TODO: use env variables or params
	@$(eval DOCKERHOST=$(shell ./get-docker-host.sh))
	@cat ./docker-compose.yml.template > ./docker-compose.yml
	awk '{sub("<IP_ADDRESS>","$(DOCKERHOST)")}1' ./docker-compose.yml > ./docker-compose.yml.tmp && \
		mv ./docker-compose.yml.tmp ./docker-compose.yml

clone:
	mkdir -p .docker
	-git clone https://github.com/bcgov/von-network .docker/von-network
	cd .docker/von-network && git reset --hard $(VON_NETWORK_VERSION)

von-up: clone
#	cd .docker/von-network && ./manage build && ./manage up
# TODO: use von-network image when ARM is supported
	docker pull ghcr.io/lauravuo/von-network:latest
	-docker tag ghcr.io/lauravuo/von-network:latest von-network-base
	cd .docker/von-network && ./manage up

# resolve docker host similarly as is done in von-network
docker-host:
	@echo "#!/bin/bash\n" > ./get-docker-host.sh
	@curl -s --raw https://raw.githubusercontent.com/bcgov/DITP-DevOps/main/code/snippets/getDockerHost >> get-docker-host.sh
	@echo "\ngetDockerHost" >> ./get-docker-host.sh
	@chmod a+x ./get-docker-host.sh

gen-cert: docker-host
	$(eval IP_ADDRESS=$(shell ./get-cert-host.sh))
	@echo $(IP_ADDRESS)
	@cat ./cert/client/conf.template > ./cert/client/cert.conf
	awk '{sub("<IP_ADDRESS>","$(IP_ADDRESS)")}1' ./cert/client/cert.conf > ./cert/client/cert.conf.tmp && \
		mv ./cert/client/cert.conf.tmp ./cert/client/cert.conf
	cd ./cert && ./gen.sh client
	@cat ./cert/server/conf.template > ./cert/server/cert.conf
	awk '{sub("<IP_ADDRESS>","$(IP_ADDRESS)")}1' ./cert/server/cert.conf > ./cert/server/cert.conf.tmp && \
		mv ./cert/server/cert.conf.tmp ./cert/server/cert.conf
	cd ./cert && ./gen.sh server

up-von:  gen-yml down von-up gen-cert ledger-ready
	@echo "see the conf/genesis.txt"

up:  gen-yml down von-up gen-cert agency
up-d:  gen-yml down von-up gen-cert agency-d

ledger-ready:
	./wait-for-ledger.sh
	curl http://localhost:9000/genesis > conf/genesis.txt

agency: ledger-ready
	docker-compose up --build

agency-d: ledger-ready
	docker-compose up --build -d

down: clone
	docker-compose down
	cd .docker/von-network && ./manage down

onboard:
	$(eval AGENT_NAME="agent-$(shell date +%s)-1")
	nohup $(FCLI) authn register --url $(AUTH_URL) \
		-u $(AGENT_NAME) \
		--key $(AUTH_KEY) \
		--origin $(AUTH_ORIGIN) > $(AGENT_NAME).log &
	sleep 1
	$(eval AGENT_NAME="agent-$(shell date +%s)-2")
	nohup $(FCLI) authn register --url $(AUTH_URL) \
		-u "$(AGENT_NAME)" \
		--key $(AUTH_KEY) \
		--origin $(AUTH_ORIGIN) > $(AGENT_NAME).log &
	sleep 1
	$(eval AGENT_NAME="agent-$(shell date +%s)-3")
	nohup $(FCLI) authn register --url $(AUTH_URL) \
		-u "$(AGENT_NAME)" \
		--key $(AUTH_KEY) \
		--origin $(AUTH_ORIGIN) > $(AGENT_NAME).log &
	sleep 1
	$(eval AGENT_NAME="agent-$(shell date +%s)-4")
	nohup $(FCLI) authn register --url $(AUTH_URL) \
		-u "$(AGENT_NAME)" \
		--key $(AUTH_KEY) \
		--origin $(AUTH_ORIGIN) > $(AGENT_NAME).log &
