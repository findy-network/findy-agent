# findy-agent

![lint](https://github.com/findy-network/findy-agent/workflows/golangci-lint/badge.svg?branch=dev)
![test](https://github.com/findy-network/findy-agent/workflows/test/badge.svg?branch=dev)

- [findy-agent](#findy-agent)
  - [About findy-agent](#about-findy-agent)
        - [Onboard Binding](#onboard-binding)
  - [Get Started](#get-started)
  - [Run The Agency](#run-the-agency)
  - [Edge Agent On-boarding](#edge-agent-on-boarding)
  - [Agency Network](#agency-network)
  - [Command-line Interface](#command-line-interface)
  - [Agency Architecture](#agency-architecture)
  - [Aries Protocol State Machine](#aries-protocol-state-machine)
  - [Missing Features For Production](#missing-features-for-production)
 
## About findy-agent

Findy-agent is a Go package and command. It implements a multi-tenant identity
agency for Aries protocols. However, it's not an Aries mediator; even it's very
similar because all of its communication protocols are DIDComm-based. It offers
a way to allocate Cloud Agent for any Edge Agent who is [bound](#onboard-binding)
to the same ecosystem.

_Please be noted that the whole Findy Agency is still under construction, and
there are many [missing features](#missing-features-for-production) for full
production use._ However, it's currently tested for an extended period of pilot
and development use, where it's proven to be stable and scalable. The current
focus of the project is to offer an efficient and straightforward multi-tenant
agency with Aries compatible agent protocols.

You can use the agency and related Go packages roughly for four purposes:

1. As a service agency for multiple Edge Agents to allocate Cloud Agents.
Allocated CAs implement [Aries agent-to-agent
protocols](#aries-protocol-state-machine) and interoperability.

1. As a [CLI tool](#command-line-interface) for setting up Edge Agent wallets,
creating schemas and credential definitions into the wallet and writing them to
the ledger. You don't need to use or install indy CLI.

1. As an admin tool to monitor and maintain an agency.

1. As a framework to implement Service Agents like issuers and verifiers.


##### Onboard Binding 
To be able to onboard, allocate an agent, the client, and the agency must
share the same `salt`. Please see the `FINDY_AGENT_SALT` environment variable,
or build your agency with the sources which set the `utils.Salt` variable.

## Get Started

1. [Install](https://github.com/hyperledger/indy-sdk/#installing-the-sdk) libindy-dev.
2. Install Go. Make sure environment variable `GOPATH`is defined.
3. Create parent folder for findy-agent-project in your [\$GOPATH](https://github.com/golang/go/wiki/SettingGOPATH):

   ```
   $GOPATH/src/github.com/findy-network
   ```

4. Clone [findy-agent](https://github.com/findy-network/findy-agent) (or move) repository to the newly created parent folder.
5. Install needed Go packages: `make deps`. This installs _findy-wrapper-go_ which is mandatory.
6. Install the command line application: `make install`
7. Verify the installation: `findy-agent -version`

    It should output:
    `OP Tech Lab - Findy Agency v. X.X`

## Run The Agency 

1. [Install and start ledger](https://github.com/bcgov/von-network/blob/master/docs/UsingVONNetwork.md#building-and-starting)
2. Create a ledger pool with the name `von`

   ```findy-agent create cnx -pool von -txn genesis.txt```
3. Go to `scripts` directory: `cd scripts`
4. Run the agency tests: `./von-network`
5. Connect to agency with your client or test it with the agency's client
command. Please see the helper scripts in the `scripts` directory.

All of that can be done with the `make scratch` as well if the predefined ledger
and steward wallet names are ok. The previous steps were for educational
purposes. If you want to start the agency fast e.g., on OSX, the Makefile
approach is preferable. Please see the scrips in the `tools` dir.


## Edge Agent On-boarding

Findy-agent serves all types of edge agents (EA) by implementing a corresponding
cloud agent (CA) for them. An EA communicates with its CA with Aries's
predecessor of DIDComm, which means that the communication between EA and CA
needs indy SKD's wallet and crypto functions. The same mechanism is used when
the agency calls a service agent (SA), a particular type of an EA which performs
as an issuer or verifier or both.

The agency offers an API to make a handshake, aka onboarding, where a new CA is
allocated and bound to an EA. findy-agent can call that same API by itself as a
client, a temporary EA. That is an easy way to onboard SAs to the agency. The
following command is an example of calling an API to make a handshake and export
the client wallet and move it where the final SA will run.

```shell script
  findy-agent client handshakeAndExport \
    -wallet ${EXPORT_NAME}_client \
    -email ${EXPORT_NAME}_server \
    -pwd ${EXPORT_KEY} \
    -url http://localhost:8080 \
    -exportpath ${EXPORT_DIR}/${EXPORT_NAME}.export
```

As you can see, that is a long command, and lots of information is needed. The
suggestion is to write these commands to owns scripts. With the findy-agent
repo, there are many scripts where to start. If more convenient CLI would be
needed, please check the `findy-agent`.

## Agency Network

findy-agent is a multi-tenant identity agency that is capable serve thousands of
edge agents with one installation, and which can scale horizontally.

The following diagram shows all the components of a typical DID/SSI-based
identity network. The server rack icon illustrates an agency. There are three in
the picture, but typically there can be as many as needed, and agencies can run
in a cluster for horizontal scalability.

![big](docs/agency-big.png?raw=true 'big')

In the middle of the picture is the indy ledger. Depending on the installation
and the type of the network, it can be a public ledger (permissioned) or just a
development ledger. All the communication to the ledger goes through the
agencies. Also, all the Aries agent-to-agent communication goes from agency
to agency, as you can see in the following drawing.

![big_aries](docs/agency-aries-big.png?raw=true 'big_aries')

The application logic is inside the edge agents which communicate and control
their cloud agents with the DIDComm-based protocol as well. The next image
illustrates when a mobile EA communicates findy-agent, it calls the agency's CA
API and receives APNS notifications, or WebSocket messages if the connection is
on.

![mobile](docs/agency-mobile.png?raw=true 'mobile')

Likewise, when a SA communicates with an agency, it calls the agency's CA API
and receives webhook calls over DIDComm from the agency. The WebSocket option
is available as well. The image below shows how CAs communicate with Aries, and
the agency notifies the SA through indy's version of DIDComm.

![sa](docs/agency-sa.png?raw=true 'sa')

## Command-line Interface

findy-agent offers an extensive set of commands by itself, and more
user-friendly command set exists in findy-agent. In addition to that, many
other tasks have to be taken care of before a full agency setup can work. The
following use case diagram shows most of the tasks and uses system boundaries to
illustrate which of them are directly operable by findy-agent or findy-agent.

![server.puml](http://www.plantuml.com/plantuml/svg/TL0nRiCm3DprYaCcUt0Ua250qQb0Dx-0aHXLjImP59qqAFBtsatGs2qw4UJTyN0N-QZG30d-JU62iDMGaobTI0C9zHZ8TkIvrKjap30b7zaOife5vFgGfcKUQ1fKXNKSK5XEBBLPhzZkKHqa98yXvuXZY5nZXv1i71sRErQKpoGEPugHzQPQ_zc1FvIJAtyC7boNRSU2KuvZltRvLz9Ir2NJ_CJ5vibpiXSylxwWGVkjtU3J08leceVwruL4vrCbF5bCzVamfRitSKCNOQxB8jc5Xs3xNdAglm00)

As the diagram shows the prerequisites to start the agency are:
 - A steward wallet is available, or you have seed to create steward wallet by
 your self.
 - You have to set up a server environment, like volumes for wallets, and
 databases.

During the agency run, you can monitor its logs. Between the starts, you can
reset the all cloud agent allocations, or you can edit the register JSON. Note,
that you cannot add new allocations only by editing the register JSON. The whole
handshake procedure must be performed.

When an agency is running, you can operate with it findy-agent executable when
you use it as a client mode. The following use case diagram shows the essential
commands after the agency started.

![client.puml](http://www.plantuml.com/plantuml/svg/VOr12i8m44NtESN7bLqKpo2w4Tnv01cIQJAqJKioeaMykv6Ag8YRuVFc_PcE6uKEIEA3mabYgp94ark98oNgCP9joVD1fuxnM5Fq7Hj3LeS4Sht4PvQSJvoECp8l5OkrvsWdRFOxrB2TSDG5hWPp6tMDPQ3eSg2MgmyyIlHVX2IT9VDgkzk2BxOKVIaLv_tzvqsKKDnnI5hz4crYKcGRkAS_GfaEZflAtEu0)

The use case diagram is a summary of the creation commands available. With these
commands, you can create all that is needed in the identity network from the
command line.

![create.puml](http://www.plantuml.com/plantuml/svg/VOv12i8m44NtESMdAxieda7ifk3E0yYGJDDWceHqH8fuTsihOa6wck6zFyFtt0ea8ZlR2OpBhCN5e8Qh2uaozKYahsJvBADdl3K5wrafqX8poFGkV7Ot33VEbmMfRnJ5mNBG8uwd1XLqPX8ky51Ohb5Ls2qKW_2Tia7TrEK_dsBqQv5Si3FUUpQMSwac_TjazLvtt5EvaPY6WU5sApENUxu0)

## Agency Architecture

findy-agent is a service that implements an Aries compatible identity agent.
What makes it an agency is that it's multitenant. It can serve thousands of edge
agents with one installation and with quite modest hardware.

Each EA gets a corresponding cloud agent as its service. The following
deployment diagram illustrates the main components of the system where
findy-agent is installed on a single node, and a wallet application is running
on a mobile device. The picture includes an external agent that is running on
another node (grey).

![main-components](http://www.plantuml.com/plantuml/svg/TLB1Yjim43rRNp5a3psjFo0i2swIBO7D4cYXb923R4ziiOh6I4fTK_BldP6TD87TGsBBR-RDUs_ag4QORQWq5c69lqs5C_YhiavNxxfXwCMuUlYfhSMOwwvBO5Rhg4iT65uKC88pW8TNqxInj2TCHTcEmKuRtvk0U_vmvjzEw4BzBkVTgjZ38-ZE1SNWMIcNr1GDkcg0DpwaSJVJB9tgJmSot-syystdwsRvTTI-stxVZEYzHv2nSQpn628hOmD9PxyQdmlH-_WCrmzRJv4giloiC0JougVm1iFwxJFSuY4AnyZzNo5pNfqz_49hgPzYh3pMOGpmIvOvYWWbnKX7e0DSK4QooB9Htj3L87KNyI03BpJi_2DXJycPX3E7Re8XH2qizqbzeob9Qqevx--sj_eJOTmW-x2oeCRhGHh6P2JN5FLUUj8To9z14fz3iLr3nHa49PS2dl8yvJGNC-PWAXqDyMLHKyHZmKJsaQUScLEjYEFOlCX9gTrVpGKTZoStyKE9iKTqm0lH7EIYK9a9q91n3SJMd_WFWMmDo_LIEdFuqUAWCbvAoucHHwamNl00aK13dnQhihurLGkziLOiGKKkQkDup03SZ5vbgOSyB6HRkRfkyXy0)

The wallet app and the agency they both include a wallet for pairwise (blue).
These wallets are used only for a pairwise between EA and CA. That makes it
possible to use DIDComm for EA/CA communication. In the future, there might be
other ways to access CA from outside of the agency. As the diagram shows, the
main wallet of each agent is on the server, our we could say it is in the cloud.
That simplifies things a lot. We can have cloud backups, recovery, and most
importantly, it makes it possible to 24/7 presence in the cloud for each agent.

The next image shows an almost identical setup, but the mobile agent is replaced
with the service agent. It's below the agency in the picture.

![sa-components](http://www.plantuml.com/plantuml/svg/TLFHQkim37sElqAahri-e8n1IBjBu386VIYNKePhAssOQnVRMM-vzD-lv6Hf0zjJefmZwUX8iKuZvCk_4SezsfZ3pBJxGznxUO5_8eFIjnZW4JO9tegh43QbSAmky4f1pamjezp9G4XbNATXBOt1iTxETCYiRBCiuIHRVsu3RaLs5Tb9gW-vfxoNrkeB_79vJpJjZZzyIngqCizZYAolAhUSzUPTTCePUYeCmVajWMc8-lKdt60J7v-74jVxKSwaTXpa3nhZphquvL4li0bzWdKHWQk0Q-26mMnzQ2CIVwKEU9GWhQQW6d1wbHwXjH0FJ1eSPNiBffLKhK7FlFAjXiPvM9OKH0VKGgR2b7aaCbeDBEAsdZg43Zsiq7-Ypq46g5SiVJIIaLRXlPKRzZRe3xRCzCAdattX16HLvprb6XACg563i-R2GAyJI6LrMpKFD8fCLy1DLkKxJKRntV7S32VDLRc6sU_90MMRQd91xF-LvqurY-8P-2Bct9nTKrGi25vjmlgESmXf8G_9I1qU0ACgnEGsuOdvasPpMDIBoXsFuoSXXjCYjdPdMup_oNU7LdGdAfaon7y0)

The issuer server could run on the same node as the agency, but the most common
case is where it runs on its own. Typical SA includes application logic, and the
issuing credential is a small part. The API between SA and the agency is quite
straightforward. The API runs on DIDComm similarly to mobile agents.

## Aries Protocol State Machine

The following sequence diagram shows an example of how two cloud agents send
messages to each other and save them to Bolt DB (Go implementation of LMBD). The
diagram shows the "transactional" implementation of HTTP-based message transfer.
Receiving the message is done by first saving the incoming message to the
database and after that returning OK. If the receiver cannot save the incoming
message, it returns an error code.

![connection-protocol-save-state.puml](http://www.plantuml.com/plantuml/svg/XP71IiGm48RlUOevMfRKRVMqBCBQzU2bhbO4JqjCWmPAKjAn-_ecSQjD6jXB8Fbcvl_8l4hi15GxGEtE0vFc90S1rr1ffGH7gHKSZ4RDTKV8dY7xO9RVwmuBqAOL1ehrclJCeEIoPmhjd8cK2rAUoOqbmR09t5f0cCqT6JgnWln6RIbrjmqCR1HNWr2jL9_7wgckZu_rMqPSABtp2QiLR81RVKU8Uw6M-91pkn5ydFKQWTz6mHTYxvLJYIScyI_nvU4v8wq8DFCyJsO5ghvnrbPwxotzzvxAGWbOYFj9iNYG3sdp9Z8llNaoBL9lid0nyPTFPMcDkNgJfMlaOE5k_tvX9SloQ1Va1m00)

The next UML diagram implements the connection protocol as a finite state
machine, which has two top-level states: Waiting Start Cmd/Msg, and
ConnectionProtocolRunning. The protocol waits for either a command or a message.
The command can be an InvitationCmd, which means that we have received an
invitation. In the current system, invitations are coming from other channels,
and they are not protocol-messages (out-bound). The invitation includes
connection information to an agent like an endpoint and its public key. The
agent who receives the InvitationCmd sends a connection request message
(ConnReq) to the receiving agent and starts to wait for a connection response
message (ConnResp). As the state machine shows, the receiving agent sends the
connection response back and finalizes its state machine. The agent who started
the connection protocol receives the connection response message and finalizes
its state machine.

![connection-psm-invitee.puml](http://www.plantuml.com/plantuml/svg/TP4nQyCm48LdwrSSMGg5uD1HGvTC7GmX7ZgK3iQ-9WBtb4Zdjj3ql_TqbetRaCdIUtVllfFPSO-m2vvzwtkekM64gccFZj2OgDVLS-FOqI6vWM7xtfLLFAoWYV0PasJCo_qhh1-dw_Y1jIYxkhBmH1zEafmdwRrojvveZsU9d0Sc2TlKC97j1o91qA7I1T_65BcuHkeINSxHaXZmF1TC-6D1F8taGKuISgVeFRwnyAIs_xY55jmmPOQeQciWM3Wodyg7pMQqB7I-Z4880x7h5wufjCC2VZak0_8GI6r8TdOrMG3i3A_FTBdSMZzl5_Ds2_OqQldKHTQk-J90_0ima_yOa_v0Diml)

The previously seen ConnectionProtocolRunning state divides two separate state
machines. The left sub-state machine is on when the agent is an inviter, and the
right-sided is on when the agent is an invitee. This same basic structure is in
all of the Aries protocol state machines. One of the agents initiates the
protocol, which gives the roles for them: inviter/invitee, issuer/holder,
prover/verifier. However, most of the Aries protocols are more complicated
because both of the roles can initiate the protocol, and depending on the
message it sends, the role it gets for the protocol.

![issue.puml](http://www.plantuml.com/plantuml/svg/VLJ1Ri8m33qtNs5nd8IATknXqiHbEw0XEEmmxL2r1YAw198K9etz-nmdfIsqEu7zulTiFuEJha9ujRQMQWlBzK88wtA7C7dFfVEvjSkDW_bNcIxiTWAvXRFrAI4-7ZvX-jI4uGEcb26Q3EO6oxVD1WsL3e9Bem_Q8h7-1uzLCxMlRVgDCr2PquMkLhLI57B3L0G_8eaFrwXAFzYLXTzOgKxo-gOPthzPuR56wyBe1e-3H5uT0v51UnWaUYxsWIGlza8al3uQYPLlzjK7uMvXIImgTMgf2wYLanNid3kaZxEPY3Wp-9Q9e8FvJ0RuBLrgqDL6CGWj69Jz72oyZHi8mY4zBkpn84nZTdGJ7pD0isNDGfXJQLgLaTkTo_WKqOHBJ0B5RQT1wN8PD28kALXne63GYjRti_PV1wcwnjkkyLscrlXpp_WkKMEylJ7UjtTtx2tCDlLlFU4Q_jLntztzHhfHPqB567NI40vls_jwREw93Eu9CzkuRhYOyNkBzsBPx1M5Mba4QWeC5YXFI4i9u8W6e-rfjNa-h0etT5SlkYff-ZMxiyYBdLGeVeKG_iyXbee_)

The issuing protocol state machine is waiting for a command to initiate the
protocol or incoming message to connect already started protocol (Waiting Start
Cmd/Msg -state). If an agent is in an issuer's role, it can start the protocol
by sending a credential offer message (CredOffer). It can do that by sending
CredOfferCmd to the protocol processor, as we can see in the state machine
diagram. Similarly, when an agent is in a holder's role, it can start the
protocol by sending a credential-propose message (CredProp), and it can do that
by sending CredProposeCmd to the protocol processor. Naturally, when an agent
receives either an offer or a propose, it responses accordingly. Receiving a
credential offer puts an agent to a holder's role, and receiving a credential
propose puts an agent to an issuer's role. Now we should understand how we have
four related ways to initiate the protocol state machine for an issuing protocol
(state transition from Waiting Start Cmd/Msg -state to
IssuingProtocolRunning-state). The rest of the protocol is quite clear and easy
to understand from the state machine diagram.


## Missing Features For Production

- [ ] Current tests run only happy paths.
- [ ] Interoperability testing with Aries testing harness.
- [ ] Indy wallet implementation with storage plugin like PostgreSQL.
- [ ] Crypto implementations for different server types, AWS Nitro, ...   
- [ ] Backup system for current docker volumes.
- [ ] The PSM runner for restarts and non-delivery messages and cleanup old ones. 
- [ ] Haven't been able to test with stable ledger.
- [ ] Check if we have received the message already.
- [ ] Check incoming order of the messages, same as declared in the PSM. 
- [x] libindy under pressure, wallet handles, etc. Done: wallet pool, more tests
      needed
- [ ] API for browsing connections, credentials etc.
- [ ] PSM archive logic, dedicated storage for persistent client data (see the
      PSM runner).
- [ ] Credential revocation, if wanted to use (check upcoming anoncreds 2.0)
- [ ] Skipping DID-writes to ledger for individuals.
- [ ] Agent permissions. Separation of individuals and services in onboarding ->
      e.g. no credential issuing for individuals (maybe Agency types).

