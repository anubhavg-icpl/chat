FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# pure-Go sqlite (modernc.org/sqlite); no CGO required
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o open_oscar_server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/open_oscar_server /app/open_oscar_server

EXPOSE 5190 8080 9898 1088 4000/udp

CMD ["/app/open_oscar_server"]
