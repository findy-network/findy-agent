run: clone agency-up tag test
run-check: clone agency-up tag test-check

prep:
	rm -rf .docker/findy-agent-backchannel/env/cert/* && \
		cp -R ../dev/docker/cert/ .docker/findy-agent-backchannel/env/cert && \
		cd .docker/findy-agent-backchannel/env && \
		make clone

test: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath-test

test-check: prep
	cd .docker/findy-agent-backchannel/env && \
		make aath-test-check

clone:
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