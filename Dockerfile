FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o babymilk ./cmd/server

# Test stage: compile test binaries, run them in the production base image
FROM builder AS test-builder
RUN CGO_ENABLED=0 go test -c -o /app/test-cmd ./cmd/server
RUN CGO_ENABLED=0 go test -c -o /app/test-app ./internal/app
RUN CGO_ENABLED=0 go test -c -tags=integration -o /app/test-app-integration ./internal/app

FROM alpine:3.19 AS test
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=test-builder /app/test-cmd .
COPY --from=test-builder /app/test-app .
COPY --from=test-builder /app/test-app-integration .
COPY --from=builder /app/static ./static

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/babymilk .
COPY --from=builder /app/static ./static
EXPOSE 8000
CMD ["./babymilk"]
