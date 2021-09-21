## TODO List

- [x]	start to figure out onboarding
- [x]	remove agency-level legacy api, handshake the first one, how about ping?
	if we leave replys to root path '/' that could serve as that and make things
	much lean
- [x]	removed code: client.go and tests
- [ ]	start to figure out protocol starters, think about `Task`
- [ ]	figure out protocol status info getters, `Task`?
- [ ]	not in this refactoring: should also SA API be async?
- [ ]	**backlog**: libindy vs Aries shared libs, Ursa?
- [ ]	replace EA did in URL with some other ID like UUID/other string 
- [ ]	try to plan how to share the workload between Harri / Laura
- [ ]	e2e tests to new findy-agent-cli which don't have libindy dependency any
	more. See the `e2e.sh` in github workload

### Background Info

- we don't need `findy-web-wallet`, but how we could build DIDs without wallet
  - something to study, can be have a temporary `DID` which is not yet in wallet
- Receiver -type. Maybe we still need that but we should rethink it.
