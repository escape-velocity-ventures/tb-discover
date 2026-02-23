FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o /tb-manage .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates util-linux iproute2
COPY --from=builder /tb-manage /tb-manage
ENTRYPOINT ["/tb-manage"]
