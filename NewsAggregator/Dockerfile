FROM golang:1.23 AS compiling_stage
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /go/src/news
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -o news_server cmd/server/main.go

FROM scratch
LABEL ver="1.0"
LABEL maintainer="zombiehunter"
WORKDIR /root/
COPY --from=compiling_stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=compiling_stage /go/src/news .
CMD ["./news_server"]