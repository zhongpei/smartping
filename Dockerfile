FROM golang:1.13

WORKDIR /go/src/app
COPY . .
RUN go mod init
env GO111MODULE=on
env GOPROXY=https://goproxy.io,direct
RUN go get -d -v ./...
RUN go install -v ./...

RUN chmod +x control
RUN ./control build

CMD ["./bin/smartping"]
