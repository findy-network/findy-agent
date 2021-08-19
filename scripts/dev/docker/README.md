# Setting up agency with von-network

This script will

1. clone von-network repository
1. launch von-network
1. build findy-agent image
1. launch built findy-agent together with auth service and connect it to von-network

Launch services:

```
make up
```

Open new terminal to onboard agents using findy-agent-cli.
Onboarding example:

```
findy-agent-cli authn register --url "http://localhost:8888" \
	-u "my-agent" \
	--key "15308490f1e4026284594dd08d31291bc8ef2aeac730d0daf6ff87bb92d4336c" \
	--origin "localhost:8888"
```