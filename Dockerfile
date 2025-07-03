FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o notificationservice cmd/*.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app/

COPY --from=build /app/notificationservice .
COPY /config/prodConfig.env /app/config/config.env

CMD ["./notificationservice"]