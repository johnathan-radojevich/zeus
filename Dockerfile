# syntax=docker/dockerfile:1

ARG GO_VERSION=1.26
FROM docker.io/library/golang:${GO_VERSION}-bookworm AS build

ENV GOTOOLCHAIN=auto

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/zeus .

FROM docker.io/library/debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=build /out/zeus /usr/local/bin/zeus

ENV TERM=xterm-256color

ENTRYPOINT ["zeus"]
