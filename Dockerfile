FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server   ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/worker   ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/webhook  ./cmd/webhook && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/migrator ./cmd/migrator && \
    mkdir -p plugins && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o plugins/generator ./cmd/generator

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

ARG CMD=server
COPY --from=builder /build/bin/${CMD} ./app
COPY --from=builder /build/plugins ./plugins/

EXPOSE 50051 8080

CMD ["./app"]
