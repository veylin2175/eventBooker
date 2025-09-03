FROM golang:1.23.6-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /event-booker ./cmd/event-booker

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /event-booker /app/event-booker

COPY ./config ./config
COPY ./static ./static

RUN chmod +x /app/event-booker

EXPOSE 8080

CMD ["/app/event-booker"]