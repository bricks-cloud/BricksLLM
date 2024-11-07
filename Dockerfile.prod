FROM golang:1.23.2 AS build
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /go/src/github.com/bricks-cloud/bricksllm/
COPY . /go/src/github.com/bricks-cloud/bricksllm/
RUN go build -ldflags="-s -w" -o ./bin/bricksllm ./cmd/bricksllm/main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /usr/bin
COPY --from=build /go/src/github.com/bricks-cloud/bricksllm/bin /go/bin
EXPOSE 8001 8002
ENTRYPOINT ["/go/bin/bricksllm"]

CMD ["-m", "production"]