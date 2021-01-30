docker-build:
	docker build -t twinproduction/gatus:latest .

build-frontend:
	npm --prefix web/app run build

run-frontend:
	npm --prefix web/app run serve