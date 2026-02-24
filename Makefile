.PHONY: up down build test run migrate seed metrics insights trends roi gaps report swagger clean

up:
	docker-compose up --build -d

down:
	docker-compose down

build:
	go build -o bin/server ./cmd/server

test:
	go test -race ./... -v

test-short:
	go test -short ./... -v

run:
	go run ./cmd/server

migrate:
	go run ./cmd/server -migrate-only

seed:
	go run ./cmd/server -seed-only

metrics:
	curl -s "http://localhost:8080/api/v1/metrics?page=1&page_size=10" | jq .

insights:
	curl -s "http://localhost:8080/api/v1/insights?page=1&page_size=20" | jq .

trends:
	curl -s "http://localhost:8080/api/v1/trends?period=MOM&metric=tpv_usd" | jq .

roi:
	curl -s "http://localhost:8080/api/v1/roi" | jq .

gaps:
	curl -s "http://localhost:8080/api/v1/market-gaps" | jq .

report:
	curl -s "http://localhost:8080/api/v1/reports/health?format=html" -o report.html && open report.html

swagger:
	swag init -g cmd/server/main.go -o docs

clean:
	docker-compose down -v
	rm -f bin/server report.html
