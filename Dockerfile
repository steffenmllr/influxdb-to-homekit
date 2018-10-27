# Stage 1: Build executable
FROM golang:1.11 as buildImage

# We start with migrate so this gets cached most of the time
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR $GOPATH/src/github.com/steffenmllr/influxdb-to-homekit
COPY . .

RUN dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o server

# Stage 2: Create release image
FROM alpine:3.6
RUN apk add --no-cache -certificates

RUN mkdir app
WORKDIR app

COPY --from=buildImage /go/src/github.com/steffenmllr/influxdb-to-homekit/server server
RUN chmod +x server

EXPOSE 12345

CMD ["/app/server"]