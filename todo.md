## TODO List

- [ ] start to figure out onboarding
- [ ] remove agency-level legacy api, handshake the first one, how about ping?
  if we leave replys to root path '/' that could serve as that and make things
  much lean
- [ ] removed code: client.go and tests
- [ ] start to figure out protocolla starters
- [ ] figure out protocolla status info getters
- [ ] not in this refactoring: should also SA API be async?
- [ ] **backlog**: libindy vs aries shared libs, ursa?
- [ ] replace AE did in url with some other ID like UUID/other string 
- [ ] try to plan how to share the workload between harri / laura

### Background Info

- we don't need `findy-web-wallet`, but how we could build DIDs without wallet
  - something to study, can be have a temporary `DID` which is not yet in wallet
- Reveiver -type. maybe we still need that but we should rethink it.
