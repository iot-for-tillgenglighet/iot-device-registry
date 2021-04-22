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

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /opt/diwise

COPY --from=builder /app/cmd/iot-device-registry/iot-device-registry /opt/diwise/

RUN chmod 775 /opt/diwise

EXPOSE 8080
USER 1001

ENTRYPOINT ["/opt/diwise/iot-device-registry"]
