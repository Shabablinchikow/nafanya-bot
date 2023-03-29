FROM golang:latest AS build

WORKDIR /go/src/github.com/shabablinchikow/nafanya-bot
COPY . /go/src/github.com/shabablinchikow/nafanya-bot

# Setup the compilation environment
ENV CGO_CPPFLAGS="-I/usr/include"
ENV CGO_LDFLAGS="-L/usr/lib -lpthread -lrt -lstdc++ -lm -lc -lgcc"
ENV CC="/usr/bin/gcc"
ENV CFLAGS="-march=x86-64"
ENV PKG_CONFIG_PATH="/usr/local/lib/pkgconfig"
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go get -d -v ./...
RUN go build -C cmd -o /go/bin/nafanya-bot

FROM gcr.io/distroless/static-debian11

COPY --from=build /go/bin/nafanya-bot /nafanya-bot
CMD ["/nafanya-bot"]