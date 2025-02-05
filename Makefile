BINARY=gatus

.PHONY: install
install:
	go build -v -o $(BINARY) .

.PHONY: run
run:
	GATUS_CONFIG_PATH=./config.yaml ./$(BINARY)

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
	docker build -t twinproduction/gatus:latest .

docker-run:
	docker run -p 8080:8080 --name gatus twinproduction/gatus:latest

docker-build-and-run: docker-build docker-run


#############
# Front end #
#############

frontend-build:
	npm --prefix web/app run build

frontend-run:
	npm --prefix web/app run serve
