FROM optechlab/indy-golang:1.16.0

ARG HTTPS_PREFIX

ENV GOPRIVATE=github.com/findy-network

WORKDIR /findy-agent

RUN git config --global url."https://"${HTTPS_PREFIX}"github.com/".insteadOf "https://github.com/"

COPY go* ./
RUN go mod download

COPY . ./
RUN make install

FROM optechlab/indy-base:1.16.0

COPY --from=0 /go/bin/findy-agent /findy-agent

ENTRYPOINT ["/findy-agent"]
