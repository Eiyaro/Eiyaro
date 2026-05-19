FROM golang:1.26-alpine AS build

RUN mkdir -p /go/src/github.com/Hoohash-Network/Eiyaro
WORKDIR /go/src/github.com/Hoohash-Network/Eiyaro

RUN apk add --no-cache git

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o eiyarowallet ./cmd/eiyarowallet

FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /go/src/github.com/Hoohash-Network/Eiyaro/eiyarowallet /app/eiyarowallet

RUN mkdir -p /nonexistent/.eiyarowallet && chown nobody:nogroup /nonexistent/.eiyarowallet && chmod 700 /nonexistent/.eiyarowallet

RUN chown nobody:nogroup /app/eiyarowallet && chmod +x /app/eiyarowallet

USER nobody
ENTRYPOINT ["/app/eiyarowallet"]