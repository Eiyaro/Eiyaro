# -- multistage docker build: stage #1: build stage
FROM golang:1.26 AS build

ENV GOEXPERIMENT=simd,jsonv2

RUN mkdir -p /go/src/github.com/eiyaro/eiyaro
WORKDIR /go/src/github.com/eiyaro/eiyaro

RUN apt-get update && apt-get install -y curl git openssh-client binutils gcc musl-dev

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -tags "deadlock pebblegozstd" -o eyarod .
RUN go build -tags "deadlock pebblegozstd" -o eiyarowallet ./cmd/eiyarowallet
RUN go build -tags "deadlock pebblegozstd" -o eiyarominer ./cmd/miner
RUN go build -tags "deadlock pebblegozstd" -o eiyaroctl ./cmd/eiyaroctl
RUN go build -tags "deadlock pebblegozstd" -o genkeypair ./cmd/genkeypair

# --- multistage docker build: stage #2: runtime image
FROM ubuntu:24.04
WORKDIR /app

RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates && \
  rm -rf /var/lib/apt/lists/*

COPY --from=build /go/src/github.com/eiyaro/eiyaro/eyarod /app/eyarod
COPY --from=build /go/src/github.com/eiyaro/eiyaro/eiyarowallet /app/eiyarowallet
COPY --from=build /go/src/github.com/eiyaro/eiyaro/eiyaroctl /app/eiyaroctl
COPY --from=build /go/src/github.com/eiyaro/eiyaro/eiyarominer /app/eiyarominer
COPY --from=build /go/src/github.com/eiyaro/eiyaro/genkeypair /app/genkeypair

RUN mkdir -p /nonexistent/.eiyarod && chown nobody:nogroup /nonexistent/.eiyarod && chmod 700 /nonexistent/.eiyarod

RUN chown nobody:nogroup /app/* && chmod +x /app/*

USER nobody
ENTRYPOINT ["/app/eyarod"]
CMD ["--utxoindex", "--saferpc"]