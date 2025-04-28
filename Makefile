build:
	go build -o bin/user-microservice ./cmd/api

test:
	go test ./...

docker-build:
	docker build -t user-microservice .

docker-run:
	docker-compose up --build
	
docker-stop:
	docker-compose down

docker-rm:
	docker rm -f user-microservice

docker-rmi:
	docker rmi -f user-microservice

docker-logs:
	docker-compose logs -f

docker-exec:
	docker exec -it user-microservice bash

docker-shell:	
	docker exec -it user-microservice sh

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

gen-docs:
	swag init -g cmd/main.go --output docs