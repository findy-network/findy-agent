/*
Package agent is a package for the cloud agent and its services. It holds all
the needed packages to implement a agency framework and the different types of
agents in it. The cloud.Agent is the most important abstraction of the package.
Other packages comm, didcomm, endp, sec, etc. offer specific services for the
cloud.Agent to be able to perform its duties like serve the connected Edge
Agent.

The Edge Agents allocate the cloud agent from the agency. Every CA has at least
one EA connected to it. The current version is not implemented for multiple EAs.
In the future, we will support multiple wallet applications per individual DID
holder.

The agent package is empty itself. All the functionality is inside sub-packages.
In the future, we might split the package into two parts. One for internal
packages (most of them) and rest for framework use (external use). There are
also few package candidates that could be moved to the first level of the
findy-agent package. Those packages are: didcomm, aries, mesg, and pltypes. They
all are didcomm message related packages. However, that is something that most
likely will not happen in the near future.

Now you can find more information about each specific
package. Summary of the packages:

 agency     encapsulates services for multi-tenancy
 aries      is implementation of Aries messages
 bus        offers implementation of notification bus for internal use
 caapi      CA API handlers, entry implementation to receive API messages
 cloud      is a package for cloud agent (CA)
 comm       communication receivers, packages, handlers and helpers
 didcomm    DIDComm messaging interfaces
 e2         err2 error type helpers
 endp       agent endpoint services to parse and calculate URLs
 handshake  onboarding services to allocate CAs for EAs
 mesg       indy agent-to-agent messages used old DIDComm, CA API, ...
 pairwise   services to make a pairwise
 pltype     payload and message types
 prot       protocol processors, state machine update, status info, notify
 psm        Protocol State Machine and Representatives save state
 sa         service agent implementation, used for integration tests
 sec        secure pipe for DIDComm transfers
 service    namespace for common and simple service.Addr aka agent endpoint
 ssi        indy specifics: DID, Agent, ledger, schema, wallet, future, ..
 status     task status helpers
 trans      secure transport implementation
 txp        transport interface for agent-to-agent connections
 utils      helpers for version, settings, salts, JSON register, nonce, ..
*/
package agent
