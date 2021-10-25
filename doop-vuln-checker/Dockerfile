FROM golang:1.17-alpine3.13 as builder
RUN apk add --no-cache make gcc musl-dev

COPY . /src
RUN make -C /src install PREFIX=/pkg GO_BUILDFLAGS='-mod vendor'

################################################################################

FROM alpine:3.13
LABEL source_repository="https://github.com/sapcc/gatekeeper-addons/tree/main/doop-vuln-checker"

RUN apk add --no-cache ca-certificates
COPY --from=builder /pkg/ /usr/

USER nobody:nobody
WORKDIR /var/empty
EXPOSE 8080
ENTRYPOINT ["/usr/bin/doop-vuln-checker"]
CMD [":8080"]
