FROM golang:latest AS build

WORKDIR /go/src/github.com/shabablinchikow/nafanya-bot
COPY . /go/src/github.com/shabablinchikow/nafanya-bot

RUN go get -d -v ./...
RUN go build -C cmd -o /go/bin/nafanya-bot

FROM gcr.io/distroless/static-debian11

COPY --from=build /go/bin/nafanya-bot /
CMD ["/nafanya-bot"]