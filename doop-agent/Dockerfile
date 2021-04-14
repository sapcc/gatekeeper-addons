FROM golang:1.16-alpine as builder
RUN apk add --no-cache make gcc musl-dev

COPY . /src
RUN make -C /src install PREFIX=/pkg GO_BUILDFLAGS='-mod vendor'

################################################################################

FROM alpine:3.13
LABEL source_repository="https://github.com/sapcc/gatekeeper-addons/tree/main/doop-agent"

RUN apk add --no-cache ca-certificates
COPY --from=builder /pkg/ /usr/

USER nobody:nobody
WORKDIR /var/empty
EXPOSE 8080
ENTRYPOINT ["/usr/bin/doop-agent"]
CMD [":8080"]
