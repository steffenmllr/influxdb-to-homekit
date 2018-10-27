# Stage 1: Build executable
FROM golang:1.11 as buildImage

# We start with migrate so this gets cached most of the time
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR $GOPATH/src/github.com/steffenmllr/influxdb-to-homekit
COPY . .

RUN dep ensure
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server

# Stage 2: Create release image
FROM alpine:3.6
RUN apk --no-cache add ca-certificates

RUN mkdir app
WORKDIR app

COPY --from=buildImage /go/src/github.com/mllrsohn/api.whatsinmymeds-pro.de/server server
RUN chmod +x server && chmod +x migrate

CMD ["/app/server"]