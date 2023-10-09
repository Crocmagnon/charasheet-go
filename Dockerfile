FROM golang:1.21 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY assets ./assets
COPY cmd ./cmd
COPY internal ./internal
COPY Makefile ./
RUN make build

CMD ["/tmp/bin/web", "-http-listen-addr", "0.0.0.0:4444"]