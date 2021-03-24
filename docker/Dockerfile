FROM golang:1.15.8-alpine as builder
RUN apk update && apk upgrade && apk add --no-cache bash libc6-compat git openssh gcc g++ librdkafka
WORKDIR /ygo-card-processor
COPY . .
RUN rm -f go.sum
RUN go build -tags musl -o ./app ./main/main.go

FROM alpine:3.13.1
WORKDIR /app
COPY --from=builder /ygo-card-processor/app .
EXPOSE 8001
CMD ["./app"]
