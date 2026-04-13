BINARY=gatus

.PHONY: install
install:
	go build -v -o $(BINARY) .

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

.PHONY: docker-build
docker-build:
	docker build -t twinproduction/gatus:latest .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 --name gatus twinproduction/gatus:latest

.PHONY: docker-build-and-run
docker-build-and-run: docker-build docker-run


#############
# Front end #
#############

.PHONY: frontend-install
frontend-install:
	npm --prefix web/app install

.PHONY: frontend-build
frontend-build:
	npm --prefix web/app run build

.PHONY: frontend-dev
frontend-dev:
	npm --prefix web/app run serve
