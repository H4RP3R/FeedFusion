FROM golang:1.23 AS compiling_stage
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /go/src/gateway
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway cmd/server/main.go

FROM scratch
LABEL ver="1.0"
LABEL maintainer="zombiehunter"
WORKDIR /root/
COPY --from=compiling_stage /go/src/gateway . 
CMD ["./gateway"]