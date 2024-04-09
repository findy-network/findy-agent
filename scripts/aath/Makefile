AGENT_DEFAULT?=acapy-main
AGENT_BOB?=findy
INCLUDE_TAGS?='@AcceptanceTest'

run: clone agency-up services-up test
run-check: clone agency-up services-up test-check

record-logs:
	mkdir -p .logs
	nohup ./record_logs.sh acme_agent &
	nohup ./record_logs.sh bob_agent &
	nohup ./record_logs.sh faber_agent &
	nohup ./record_logs.sh mallory_agent &

prep: record-logs
	rm -rf .docker/findy-agent-backchannel/env/cert/* && \
		cp -a ../dev/docker/cert/. .docker/findy-agent-backchannel/env/cert/ && \
		cd .docker/findy-agent-backchannel/env && \
		make clone

# included tags can be defined with INCLUDE_TAGS
# by default tags are defined in findy-agent-backchannel
test: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath \
			AGENT_DEFAULT=$(AGENT_DEFAULT) \
			AGENT_BOB=$(AGENT_BOB) \
			INCLUDE_TAGS=$(INCLUDE_TAGS)

# check returns failure exit code, suitable for CI
test-check: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath-check \
			AGENT_DEFAULT=$(AGENT_DEFAULT) \
			AGENT_BOB=$(AGENT_BOB)

build:
	cd ../dev/docker && \
		docker compose down && \
		docker compose up --build -d

build-test: build test

# test always with latest findy-AATH-bundle from master
clone:
	-rm -rf .docker/findy-agent-backchannel
	mkdir -p .docker
	-git clone https://github.com/findy-network/findy-agent-backchannel.git .docker/findy-agent-backchannel

agency-up:
	cd ../dev/docker && \
		make up-d

services-up:
	cd .docker/findy-agent-backchannel/env && \
		make clone && \
		make tails-up && \
		make resolver-up

down:
	cd ../dev/docker && make down
	cd .docker/findy-agent-backchannel/env/.docker/aries-agent-test-harness && \
		./manage service stop von-network && ./manage service clean von-network && \
		./manage service stop uniresolver && ./manage service clean uniresolver && \
		./manage service stop indy-tails && ./manage service clean indy-tails

