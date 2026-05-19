FROM golang:1.26-alpine AS build

RUN mkdir -p /go/src/github.com/Hoohash-Network/Eiyaro
WORKDIR /go/src/github.com/Hoohash-Network/Eiyaro

RUN apk add --no-cache git

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o eiyaroctl ./cmd/eiyaroctl

FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /go/src/github.com/Hoohash-Network/Eiyaro/eiyaroctl /app/eiyaroctl

RUN chown nobody:nogroup /app/eiyaroctl && chmod +x /app/eiyaroctl

USER nobody
ENTRYPOINT ["/app/eiyaroctl"]