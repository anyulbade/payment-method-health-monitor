FROM golang:1.22-alpine AS builder

RUN apk --no-cache add git

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server .
COPY migrations/ ./migrations/
COPY docs/ ./docs/

EXPOSE 8080

CMD ["./server"]
