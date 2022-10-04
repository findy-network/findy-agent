# Interoperability testing with AATH

This folder contains scripts to test findy-agent interoperability with other Aries agents.

The script will pull/build and launch several containers:

- von-network ledger with each node in separate container
- agency auth container based on latest auth image
- agency core container built from the current branch (sources) of this repository
- agency backchannel
- acapy backchannel
- test harness

Run the default test set by executing:

```sh
make run
```

By default findy is playing the role of Bob and acapy handles rest of the roles.
You can change this using the AGENT_DEFAULT and AGENT_BOB settings:

```sh
make run AGENT_DEFAULT=findy AGENT_BOB=acapy
```

If you wish to run only specific tests, you can set the tags with INCLUDE_TAGS setting:

```sh
make run AGENT_DEFAULT=findy AGENT_BOB=acapy INCLUDE_TAGS='@T001-RFC0160'
```

## CI

Interoperability tests are run each time PRs are merged to dev.
Logs are recorded from each agent and core agency. Those can be downloaded through GitHub UI.