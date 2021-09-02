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
		cp -R ../dev/docker/cert/ .docker/findy-agent-backchannel/env/cert && \
		cd .docker/findy-agent-backchannel/env && \
		make clone

test: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath-test INCLUDE_TAGS='@T001-RFC0160,@T001-RFC0036,@T001-RFC0037'

test-check: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath-test-check INCLUDE_TAGS='@T001-RFC0160,@T001-RFC0036,@T001-RFC0037'

# test always with latest findy-AATH-bundle from master
clone:
	-rm -rf .docker/findy-agent-backchannel
	mkdir -p .docker
	-git clone https://github.com/findy-network/findy-agent-backchannel.git .docker/findy-agent-backchannel && \
		cd .docker/findy-agent-backchannel && git checkout build-images

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