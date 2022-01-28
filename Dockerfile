# Build the binary
FROM golang:1.17-alpine AS builder

WORKDIR /usr/src/app

COPY go.* ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tls-secret-injector .

# Create the final image containing only the binary
FROM scratch
USER nobody:nogroup

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /usr/src/app/tls-secret-injector /tls-secret-injector

ENTRYPOINT ["/tls-secret-injector"]
