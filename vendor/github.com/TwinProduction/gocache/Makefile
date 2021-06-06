db: docker-build
dr: docker-run
drmem: docker-run-max-memory-usage

docker-build:
	docker build --tag=gocache-server .

docker-run:
	docker run -p 6666:6379 -e AUTOSAVE=true -e MAX_CACHE_SIZE=0 --name gocache-server -d gocache-server

docker-run-max-memory-usage:
	docker run -p 6666:6379 -e AUTOSAVE=true -e MAX_CACHE_SIZE=0 -e MAX_MEMORY_USAGE=524288000 --name gocache-server -d gocache-server

run:
	PORT=6666 go run cmd/server/main.go

start-redis:
	docker run -p 6379:6379 --name redis -d redis

redis-benchmark:
	redis-benchmark -p 6666 -t set,get -n 10000000 -r 200000 -q -P 512 -c 512

memtier-benchmark:
	memtier_benchmark --port 6666 --hide-histogram --key-maximum 100000 --ratio 1:1 --expiry-range 1-100 --key-pattern R:R --randomize -n 100000