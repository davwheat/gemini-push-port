FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY src .

RUN go build -o pushport main.go

FROM alpine:latest

RUN apk add --update ca-certificates # Certificates for SSL
RUN apk add --update tzdata # Timezone data

WORKDIR /root/

COPY --from=builder /app/pushport .

ENTRYPOINT ["./pushport"]
