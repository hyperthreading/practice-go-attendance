FROM golang:1.21-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN go build -o /app/bin/main ./cmd/api

WORKDIR /dist

RUN cp /app/bin/main .

FROM debian:bookworm

RUN apt update
RUN apt install -y curl

COPY --from=builder /dist/main /main

ENTRYPOINT [ "/main" ]