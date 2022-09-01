FROM golang:1.14.9 as builder

ARG GITHUB_TOKEN
ENV GITHUB_AUTH_URL="https://${GITHUB_TOKEN}:x-oauth-basic@github.com/Semantics3"


RUN echo $GITHUB_AUTH_URL
WORKDIR /go/src/github.com/Semantics3/go-crawler
ADD . /go/src/github.com/Semantics3/go-crawler

# Re-write github url to use access token auth
RUN git config --global url.$GITHUB_AUTH_URL.insteadOf https://github.com/Semantics3

RUN make install
RUN make build

FROM alpine:latest

RUN apk update && apk add --no-cache curl openssl openssl-dev bash file iputils
RUN apk add ca-certificates && update-ca-certificates

COPY --from=builder /go/src/github.com/Semantics3/go-crawler/config /code/config
COPY --from=builder /go/src/github.com/Semantics3/go-crawler/bin /code/bin
COPY --from=builder /go/src/github.com/Semantics3/go-crawler/go-crawler /code/go-crawler

WORKDIR /code

CMD ["./go-crawler", "1>>$LOGFILE", "2>&1"]
