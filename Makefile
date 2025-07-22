generate:
	go generate ./...

docker-build:
	docker build -t Bamboo/gomodd:v1.22.1 .

docker-start-env:
	docker-compose -f docker-compose-env.yaml up -d

docker-start-server:
	docker-compose -f docker-compose.yaml up -d

docker-stop-server:
	docker-compose -f docker-compose.yaml down

docker-stop-env:
	docker-compose -f docker-compose-env.yaml down

docker-net-remove:
	docker network rm cloudOps_net

dev: docker-build docker-start-env docker-start-server

stop: docker-stop-env docker-stop-server docker-net-remove