AGENT_DEFAULT?=acapy
AGENT_BOB?=findy
INCLUDE_TAGS?='@T001-RFC0160,@T001-RFC0036,@T001-RFC0037'

run: clone agency-up tag test
run-check: clone agency-up tag test-check

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
		make aath-test \
			AGENT_DEFAULT=$(AGENT_DEFAULT) \
			AGENT_BOB=$(AGENT_BOB) \
			INCLUDE_TAGS=$(INCLUDE_TAGS)

# check returns failure exit code, suitable for CI
test-check: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath-test-check \
			AGENT_DEFAULT=$(AGENT_DEFAULT) \
			AGENT_BOB=$(AGENT_BOB)

# test always with latest findy-AATH-bundle from master
clone:
	-rm -rf .docker/findy-agent-backchannel
	mkdir -p .docker
	-git clone https://github.com/findy-network/findy-agent-backchannel.git .docker/findy-agent-backchannel

agency-up:
	cd ../dev/docker && \
		make up-d

tag:
	docker pull ghcr.io/findy-network/findy-agent-backchannel:findy-bc-latest
	docker tag ghcr.io/findy-network/findy-agent-backchannel:findy-bc-latest findy-agent-backchannel:latest
	docker pull ghcr.io/findy-network/findy-agent-backchannel:acapy-bc-latest
	docker tag ghcr.io/findy-network/findy-agent-backchannel:acapy-bc-latest acapy-agent-backchannel:latest
	docker pull ghcr.io/findy-network/findy-agent-backchannel:aath-latest
	docker tag ghcr.io/findy-network/findy-agent-backchannel:aath-latest aries-test-harness:latest

down:
	cd ../dev/docker && \
		make down