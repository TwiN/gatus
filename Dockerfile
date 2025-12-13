# Build the go application into a binary
FROM golang:alpine AS builder

ARG VERSION=dev
ARG COMMIT_HASH=unknown

RUN apk --update add ca-certificates
WORKDIR /app
COPY . ./
RUN go mod tidy -diff
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
-ldflags "-X github.com/TwiN/gatus/v5/buildinfo.version=${VERSION} \
-X github.com/TwiN/gatus/v5/buildinfo.commitHash=${COMMIT_HASH} \
-X github.com/TwiN/gatus/v5/buildinfo.time=$(date -u +'%Y-%m-%dT%H:%M:%SZ' || echo 'unknown')" \
-o gatus .

# Run Tests inside docker image if you don't have a configured go environment
#RUN apk update && apk add --virtual build-dependencies build-base gcc
#RUN go test ./... -mod vendor

# Run the binary on an empty container
FROM scratch
COPY --from=builder /app/gatus .
COPY --from=builder /app/config.yaml ./config/config.yaml
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV GATUS_CONFIG_PATH=""
ENV GATUS_LOG_LEVEL="INFO"
ENV PORT="8080"
EXPOSE ${PORT}
ENTRYPOINT ["/gatus"]
