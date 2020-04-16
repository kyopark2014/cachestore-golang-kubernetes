###################
##  build stage  ##
###################
FROM golang:1.13.0-alpine as builder
WORKDIR /cachestore-golang-kubernetes
COPY . .
RUN go build -v -o cachestore-golang-kubernetes

##################
##  exec stage  ##
##################
FROM alpine:3.10.2
WORKDIR /app
COPY ./configs/config.json.default ./configs/config.json
COPY --from=builder /cachestore-golang-kubernetes /app/
CMD ["./cachestore-golang-kubernetes"]
