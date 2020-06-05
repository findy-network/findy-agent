FROM optechlab/indy-golang:1.14.2

WORKDIR /go/src/github.com/optechlab/findy-agent

ADD .docker/findy-go /go/src/github.com/optechlab/findy-go
ADD . .

RUN make deps && make install

FROM optechlab/indy-base:1.14.2

ADD ./tools/start-server.sh /start-server.sh

COPY --from=0 /go/bin/findy-agent /findy-agent

RUN echo "{}" > /root/findy.json

EXPOSE 8080

ENV HOST_ADDR localhost
ENV REGISTRY_PATH /root/findy.json
ENV PSMDB_PATH /root/findy.bolt
ENV FINDY_AGENT_CERT_PATH /aps.p12

CMD ["/start-server.sh", "/"]