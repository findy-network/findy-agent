## TODO List

- [x] start to figure out onboarding
- [x] remove agency-level legacy API, handshake the first one, how about ping?
      if we leave replys to root path '/' that could serve as that and make things
      much lean
- [x] removed code: client.go and tests
- [x] check how pairwise is built, if we need it only for Aries, rewrite or
      simplyfie 

- [ ] check how CA endpoint is built, simplify
- [ ] Refactor SA API that it won't use old `mesg` package anymore
- [ ] remove SA API plug-in system
- [ ] WebSocket notification call remove 
- [ ] APNS notification remove
- [ ] Redesign status queries for PSMs. 
- [ ] protocol (PSM) engine uses old messages because of the legacy API
- [ ] should we add protocol implementation type ID to our API like indy/w3c?
Should it be in `Protocol` interface or settings? Maybe in `Protocol` if we can
speak many protocols in the same installation.

- [ ] Harri will continue previous cleanup, e.g. with `mesg` package

- [ ] start to figure out protocol starters, think about `Task`, Laura

- [ ] figure out protocol status info getters, `Task`?, Laura

- [ ] not in this refactoring: should also SA API be async?
- [ ] replace EA did in URL with some other ID like UUID/other string 
- [ ] try to plan how to share the workload between Harri / Laura
- [ ] e2e tests to new findy-agent-cli which don't have libindy dependency any
      more. See the `e2e.sh` in github workload
- [ ] logs are leaking secrets
- [ ] Reps aren't crypted, 
- [ ] Logging Aries messages, check log secrets, check tools!!

- [ ] **backlog**: libindy vs Aries shared libs, Ursa?


### Background Info

- Receiver -type. Maybe we still need that but we should rethink it.
