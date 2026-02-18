FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o /tb-discover .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates util-linux iproute2
COPY --from=builder /tb-discover /tb-discover
ENTRYPOINT ["/tb-discover"]
