FROM golang:1.27-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY . .


RUN CGO_ENABLED=0 GOOS=linux go build -o /hotel-booking .

FROM alpine:3.19


COPY --from=builder /hotel-booking /hotel-booking

EXPOSE 8080
ENTRYPOINT ["/hotel-booking"]