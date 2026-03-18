BINARY=gatus
VERSION=$(shell git describe --tags --exact-match 2> /dev/null)

.PHONY: install
install:
	go build -v -ldflags "-X github.com/TwiN/gatus/v5/buildinfo.version=$(VERSION)" -o $(BINARY) .

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

DIRTY=$(shell test -n "$$(git status --porcelain)" && echo "-dirty")
docker-build:
	docker build --build-arg VERSION=$(VERSION) \
		--build-arg REVISION=$(shell git rev-parse HEAD)$(DIRTY) \
		--build-arg REVISION_DATE=$(shell TZ=UTC git show -s --format=%cd --date=iso-strict-local) \
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
