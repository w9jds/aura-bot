FROM golang:latest as builder

WORKDIR /go/src/aura-bot

COPY . /go/src/aura-bot

RUN go get -d ./...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-w -s' -a -installsuffix cgo -o bot
RUN curl -o ca-certificates.crt https://raw.githubusercontent.com/bagder/ca-bundle/master/ca-bundle.crt

FROM scratch

WORKDIR /go/src/aura-bot

COPY --from=builder /go/src/aura-bot/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/aura-bot/bot /go/src/aura-bot

CMD ["./bot"]