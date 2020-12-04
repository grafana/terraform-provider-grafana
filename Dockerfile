FROM golang:1.14

ENV GOFLAGS=-mod=readonly

WORKDIR /go/src/github.com/terraform-providers/terraform-provider-grafana

COPY . .
