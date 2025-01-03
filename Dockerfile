FROM golang:alpine
WORKDIR /build
COPY ca.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates
COPY main.go .
RUN go build -o gitcom main.go
CMD ["./gitcom"]