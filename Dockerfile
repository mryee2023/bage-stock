FROM golang:1.20 AS buildStage
WORKDIR /go/src/app
COPY . .
RUN go build -o ./main src/main.go
FROM alpine:latest
WORKDIR /app
COPY --from=buildStage /go/src/app/main /app/
ENTRYPOINT ./main