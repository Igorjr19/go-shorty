build:
	go build -o bin/server cmd/api/main.go
	go build -o bin/migrate cmd/migrate/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./...

migrate-up:
	go run cmd/migrate/main.go -direction=up

migrate-down:
	go run cmd/migrate/main.go -direction=down

migrate-up-step:
	go run cmd/migrate/main.go -direction=up -steps=$(or $(STEPS),1)

migrate-down-step:
	go run cmd/migrate/main.go -direction=down -steps=$(or $(STEPS),1)

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart:
	docker compose restart

docker-clean:
	docker compose down -v --rmi all

docker-migrate-up:
	docker exec go-shorty-app ./migrate -direction=up

docker-migrate-down:
	docker exec go-shorty-app ./migrate -direction=down

docker-migrate-up-step:
	docker exec go-shorty-app ./migrate -direction=up -steps=$(or $(STEPS),1)

docker-migrate-down-step:
	docker exec go-shorty-app ./migrate -direction=down -steps=$(or $(STEPS),1)

clean:
	rm -rf bin/
	go clean
