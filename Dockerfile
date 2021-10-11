FROM golang:1.17.2 AS build

COPY cmd/ /build/cmd/
COPY internal/ /build/internal/
COPY go.mod /build/go.mod
COPY go.sum /build/go.sum

WORKDIR /build/

RUN go build -o octoflux ./cmd/octoflux

FROM debian:stable-slim

RUN apt update
RUN apt install -y ca-certificates

COPY --from=build /build/octoflux /octoflux

ENTRYPOINT [ "/octoflux" ]
