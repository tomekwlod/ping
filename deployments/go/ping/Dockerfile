# First step - just building the go app
FROM golang:1.9.7 as builder
ENV WORKDIR /go/src/app
WORKDIR ${WORKDIR}
COPY . .
RUN go get -u github.com/golang/dep/cmd/dep \
    && dep init && dep ensure \
    && cd ${WORKDIR}/cmd/ping \
    && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ping .

# Second step - copying the files and running the exec
FROM alpine:3.8
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /go/src/app/cmd/ping/ping .
COPY --from=builder /go/src/app/configs/parameters.yml configs/parameters.yml

# ENV CONFIG_PATH /root/configs

CMD ["./ping"]