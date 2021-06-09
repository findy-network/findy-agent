## Scripts

## Publishing new version

Release script will tag the current version and push the tag to remote. This
will trigger e2e-tests in CI automatically and if they succeed, the tag is
merged to master.

Release script assumes it is triggered from dev branch. It takes one parameter,
the next working version. E.g. if current working version is 0.1.0, following
will release version 0.1.0 and update working version to 0.2.0.

```bash
git checkout dev
./release 0.2.0
```

## Running e2e tests

Run end-to-end tests for findy-agent with:

```
make e2e
```

This starts test-ledger & runs e2e tests for findy-agent.

`make e2e_ci` doesn't initialize test-ledger.
