# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tracking ./cmd/tracking

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/tracking /tracking
EXPOSE 8002
USER nonroot:nonroot
ENTRYPOINT ["/tracking"]
