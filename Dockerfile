FROM golang:latest as build

RUN mkdir -p /go/src/github.com/nicholasjackson/consul-release-controller

COPY . /go/src/github.com/nicholasjackson/consul-release-controller/

WORKDIR /go/src/github.com/nicholasjackson/consul-release-controller

RUN go get ./... && CGO_ENABLED=0 GOOS=linux go build -o /bin/consul-release-controller ./cmd


FROM alpine:latest

RUN apk --update add ca-certificates

COPY --from=build /bin/consul-release-controller /bin/consul-release-controller

ENTRYPOINT ["/bin/consul-release-controller"]
