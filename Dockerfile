FROM golang:1.22.6-alpine AS builder

WORKDIR /azalbot

COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./main.go

RUN go build -ldflags "-s -w" -o azal-bot

FROM gcr.io/distroless/static-debian12:latest

WORKDIR /azalbot

COPY --from=builder /azalbot/azal-bot /azalbot/azal-bot

ENTRYPOINT ["./azal-bot"]