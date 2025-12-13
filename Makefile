BINARY=gatus
VERSION=$(shell git describe --tags --exact-match || echo "dev")
COMMIT_HASH=$(shell git rev-parse HEAD)
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: install
install:
	go build -v -ldflags "-X github.com/TwiN/gatus/v5/buildinfo.version=$(VERSION) \
		-X github.com/TwiN/gatus/v5/buildinfo.commitHash=$(COMMIT_HASH) \
		-X github.com/TwiN/gatus/v5/buildinfo.time=$(BUILD_TIME)" -o $(BINARY) .

.PHONY: run
run:
	ENVIRONMENT=dev GATUS_CONFIG_PATH=./config.yaml go run main.go

.PHONY: run-binary
run-binary:
	ENVIRONMENT=dev GATUS_CONFIG_PATH=./config.yaml ./$(BINARY)

.PHONY: clean
clean:
	rm $(BINARY)

.PHONY: test
test:
	go test ./... -cover


##########
# Docker #
##########

docker-build:
	docker build --build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		-t twinproduction/gatus:latest .

docker-run:
	docker run -p 8080:8080 --name gatus twinproduction/gatus:latest

docker-build-and-run: docker-build docker-run


#############
# Front end #
#############

frontend-install-dependencies:
	npm --prefix web/app install

frontend-build:
	npm --prefix web/app run build

frontend-run:
	npm --prefix web/app run serve
