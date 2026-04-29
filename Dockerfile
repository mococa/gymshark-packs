FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build tools (pinned versions match go.mod)
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001 && \
    go install github.com/swaggo/swag/cmd/swag@v1.16.3

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files for code generation
COPY cmd ./cmd
COPY internal ./internal
COPY docs ./docs

# Generate code and build binary
RUN templ generate && \
    swag init -g cmd/server/main.go -o docs && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# prod: AWS Lambda base with Web Adapter for production deployment
FROM public.ecr.aws/lambda/provided:al2023 AS prod
COPY --from=builder /app/server /var/task/server
COPY --from=public.ecr.aws/awsguru/aws-lambda-adapter:0.8.1 /lambda-adapter /opt/extensions/lambda-adapter
ENV PORT=8080
CMD ["/var/task/server"]

# dev: plain Alpine image for local docker run / docker-compose (default)
FROM alpine:3.20 AS dev
WORKDIR /app
COPY --from=builder /app/server .
ENV PORT=8080
EXPOSE 8080
CMD ["./server"]
