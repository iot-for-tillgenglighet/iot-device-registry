FROM golang:1.16 AS builder


# Set the Current Working Directory inside the container
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go test -v ./...

WORKDIR /app/cmd/iot-device-registry

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine

RUN apk add ca-certificates

COPY --from=builder /app/cmd/iot-device-registry/iot-device-registry /app/

ENTRYPOINT ["/app/iot-device-registry"]
