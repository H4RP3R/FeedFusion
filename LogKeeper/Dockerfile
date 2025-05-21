FROM golang:1.23 AS compiling_stage
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /go/src/logkeeper
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -o logkeeper ./service.go

FROM scratch
LABEL ver="1.0"
LABEL maintainer="zombiehunter"
WORKDIR /root/
COPY --from=compiling_stage /go/src/logkeeper .
ENTRYPOINT ["./logkeeper"]
