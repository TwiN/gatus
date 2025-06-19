#Build static webUI
FROM node:16-alpine AS node_builder
WORKDIR /app
COPY ./ ./
RUN cd /app/web/app && \
    npm install && \
    npm run build

# Build the go application into a binary
FROM golang:alpine AS go_builder
RUN apk --update add ca-certificates
WORKDIR /app
COPY --from=node_builder /app/ ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gatus .

# Run Tests inside docker image if you don't have a configured go environment
#RUN apk update && apk add --virtual build-dependencies build-base gcc
#RUN go test ./... -mod vendor

# Run the binary on an empty container
FROM scratch
COPY --from=go_builder /app/gatus .
COPY --from=go_builder /app/config.yaml ./config/config.yaml
COPY --from=go_builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV GATUS_CONFIG_PATH=""
ENV GATUS_LOG_LEVEL="INFO"
ENV PORT="8080"
EXPOSE ${PORT}
ENTRYPOINT ["/gatus"]
