set -e

make down
make run-check AGENT_DEFAULT=acapy AGENT_BOB=findy
make test-check AGENT_DEFAULT=findy AGENT_BOB=acapy
make test-check AGENT_DEFAULT=javascript AGENT_BOB=findy
make test-check AGENT_DEFAULT=findy AGENT_BOB=javascript
make test-check AGENT_DEFAULT=findy AGENT_BOB=findy