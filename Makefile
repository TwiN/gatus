docker-build:
	docker build -t twinproduction/gatus:latest .

docker-build-and-run:
	docker build -t twinproduction/gatus:latest . && docker run -p 8080:8080 --name gatus twinproduction/gatus:latest

build-frontend:
	npm --prefix web/app run build

run-frontend:
	npm --prefix web/app run serve

test:
	go test -mod=vendor ./... -cover