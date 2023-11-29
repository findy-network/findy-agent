FROM ghcr.io/findy-network/findy-base:indy-1.16.ubuntu-22.04 AS indy-base

FROM golang:1.21-bookworm AS agent-builder

# install indy deps and files from base
RUN apt-get update && apt-get install -y libsodium23 libzmq5

COPY --from=indy-base /usr/include/indy /usr/include/indy
COPY --from=indy-base /usr/lib/libindy.a /usr/lib/libindy.a
COPY --from=indy-base /usr/lib/libindy.so /usr/lib/libindy.so

# manually copy libssl1.1 as it is no longer supported for newer ubuntu
COPY --from=indy-base /usr/lib/engines-1.1 /usr/lib/engines-1.1
COPY --from=indy-base /usr/lib/libcrypto.so.1.1 /usr/lib/libcrypto.so.1.1
COPY --from=indy-base /usr/lib/libssl.so.1.1 /usr/lib/libssl.so.1.1

# linter
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

ENV GO_TOOLS="\
  github.com/cweill/gotests/gotests@latest \
  github.com/fatih/gomodifytags@latest \
  github.com/josharian/impl@latest \
  github.com/haya14busa/goplay/cmd/goplay@latest \
  github.com/go-delve/delve/cmd/dlv@latest \
  honnef.co/go/tools/cmd/staticcheck@latest \
  golang.org/x/tools/gopls@latest"

RUN echo "${GO_TOOLS}" | xargs -n 1 go install -v

ENV CI true
