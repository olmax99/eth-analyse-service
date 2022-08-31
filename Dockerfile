FROM golang:1.18 AS builder
WORKDIR /go/src/github.com/eth-analyse-service
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -v -o eth-analyse ./cmd/main.go 

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/eth-analyse-service/eth-analyse ./
CMD ["./eth-analyse"]
