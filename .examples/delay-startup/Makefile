BINARY=gatus

# Because there's a folder called "test", we need to make the target "test" phony
.PHONY: test

install:
	go build -mod vendor -o $(BINARY) .

run:
	GATUS_CONFIG_PATH=./config.yaml ./$(BINARY)

delay_and_run:
	GATUS_CONFIG_PATH=./config.yaml GATUS_DELAY_START_SECONDS=5 ./$(BINARY)

clean:
	rm $(BINARY)

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
