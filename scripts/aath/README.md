# Interoperability testing with AATH

This folder contains scripts to test findy-agent interoperability with other Aries agents.

Run the default test set by executing:
```
make run-check
```

The script will pull/build and launch several containers:
- von-network ledger with each node in separate container
- agency auth container based on latest auth image
- agency core container built from the current branch (sources) of this repository
- agency backchannel
- acapy backchannel
- test harness

TODO:
- which agents play which role
- test tags setting
