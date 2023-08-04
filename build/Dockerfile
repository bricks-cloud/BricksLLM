FROM golang:1.19 AS build
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /go/src/github.com/bricks-cloud/BricksLLM/
COPY . /go/src/github.com/bricks-cloud/BricksLLM/
RUN go build -ldflags="-s -w" -o ./bin/bricksllm ./cmd/bricksllm/main.go

FROM alpine:3.17
RUN apk --no-cache add ca-certificates
WORKDIR /usr/bin
COPY --from=build /go/src/github.com/bricks-cloud/BricksLLM/bin /go/bin
EXPOSE 8080
ENTRYPOINT ["/go/bin/bricksllm"]

CMD ["-c", "/bricksllm.yaml", "-m", "production"]