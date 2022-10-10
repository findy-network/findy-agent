/*
Package main is an application package for Findy Agency Service. Please be noted
that the whole Findy Agency is still under construction, and there are many
missing features for full production use. Also there are some refactoring
candidates for restructuring the internal package layout. However, the
findy-agent is currently tested for an extended period of pilot and development
use, where it's proven to be stable. The current focus of the project is to
offer efficient and straightforward multi-tenant agency with Aries compatible
agent protocols.

You can use the agency and related Go packages roughly for four purposes:

1. As a service agency for multiple Edge Agents at the same time by implementing
them corresponding Cloud Agents. Allocated CAs implement Aries agent to agent
protocols and interoperability.

2. As a CLI tool for setting up Edge Agent wallets, creating schemas and
credential definitions into the wallet and writing them to the ledger. You can
use findy-agent's own CLI for most of the needed tasks but for be usability we
recommend to use findy CLI.

3. As an admin tool to monitor and maintain agency.

4. As a framework to implement Service Agents like issuers and verifiers. There
are Go helpers to onboard EAs to agency and the Client, which hides the
connections to the agency.

# About the build-in CLI

The agency's compilation includes command and flag sets to operate it with
minimal dependencies to other repos or utilities. The offered command set is
minimal, but it offers everything to set up and maintain an agency. There is a
separate CLI UI in other repo, which includes an extended command set with
auto-completion scripts.

# Documentation

The whole codebase is still heavily under construction, but the main principles
are ready and ok. Documentation is very minimal and partially missing.

# Sub-packages

findy-agent can be used as a service, as a framework and a CLI tool. It's
structured to the following sub-packages:

	agent    includes framework packages like agency, agents, didcomm, endp, ..
	client   is a package to connect the agency from remote
	enclave  implements a secure enclave (interfaces) for the server
	protocol includes processors for Aries agent-to-agent protocols
	server   implements the http server for APIs, endpoints, etc.
	std      a root package for Aries protocol messages
*/
package main
