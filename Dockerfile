# ---- build stage: has the Go toolchain ----
# Keep this tag in sync with the "go" line in your go.mod.
FROM golang:1.25-alpine AS build
WORKDIR /src

# Copy dependency files first so this layer is cached until they change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO off = fully static binary, so it runs on a minimal image.
RUN CGO_ENABLED=0 go build -o /monitor .

# ---- final stage: just the binary ----
FROM alpine:3.20
# CA certificates are required, or every HTTPS check fails with a TLS error.
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app
USER app
COPY --from=build /monitor /monitor
EXPOSE 8080
ENTRYPOINT ["/monitor"]
