FROM golang:1.14.0-alpine3.11 AS builder

RUN apk add --update --no-cache make bash git openssh-client build-base musl-dev curl wget

ADD . /src/app

WORKDIR /src/app

RUN mkdir ./bin && \
    go build -o ./bin/postmanq -a cmd/postmanq.go && \
    go build -o ./bin/pmq-grep -a cmd/pmq-grep.go && \
    go build -o ./bin/pmq-publish -a cmd/pmq-publish.go && \
    go build -o ./bin/pmq-report -a cmd/pmq-report.go

FROM alpine:3.11

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip
COPY --from=builder /src/app/bin/postmanq /bin/postmanq
COPY --from=builder /src/app/bin/pmq-grep /bin/pmq-grep
COPY --from=builder /src/app/bin/pmq-publish /bin/pmq-publish
COPY --from=builder /src/app/bin/pmq-report /bin/pmq-report
COPY *.pem /etc/
COPY example.config.yaml /etc/postmanq.yaml

ENTRYPOINT ["postmanq"]
CMD ["-f", "/etc/postmanq.yaml"]