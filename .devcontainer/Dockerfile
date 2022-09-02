FROM ghcr.io/findy-network/findy-base:indy-1.16.ubuntu-18.04 AS indy-base

FROM golang:1.18-buster AS agent-builder

ENV INDY_LIB_VERSION="1.16.0"

# install indy deps and files from base
RUN apt-get update && apt-get install -y libsodium23 libssl1.1 libzmq5
COPY --from=indy-base /usr/include/indy /usr/include/indy
COPY --from=indy-base /usr/lib/libindy.a /usr/lib/libindy.a
COPY --from=indy-base /usr/lib/libindy.so /usr/lib/libindy.so

# linter
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.48.0

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
