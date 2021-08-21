all: new-install test

new-install:
	@go get -v -u ./...

generate:
	@go generate ./...

test:
	@go test ./...