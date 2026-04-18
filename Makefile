.PHONY: proto build run test test-integration test-all clean docker-up docker-down docker-build docker-logs deps

proto:
	protoc --go_out=. --go_opt=module=zxc \
	  --go-grpc_out=. --go-grpc_opt=module=zxc \
	  proto/user.proto
	protoc --go_out=. --go_opt=module=zxc \
	  --go-grpc_out=. --go-grpc_opt=module=zxc \
	  proto/tenant.proto
	protoc --go_out=. --go_opt=module=zxc \
	  --go-grpc_out=. --go-grpc_opt=module=zxc \
	  proto/release.proto
	protoc --go_out=. --go_opt=module=zxc \
	  --go-grpc_out=. --go-grpc_opt=module=zxc \
	  proto/target.proto
	protoc --go_out=. --go_opt=module=zxc \
	  --go-grpc_out=. --go-grpc_opt=module=zxc \
	  proto/payload.proto

build: proto
	go build -o bin/server cmd/server/main.go
	go build -o bin/worker cmd/worker/main.go
	go build -o bin/migrate cmd/migrator/main.go

run: build
	./bin/server -config config.toml

run-worker: build
	./bin/worker -config config.toml

migrate: build
	./bin/migrate -config config.toml

migrate-dry-run: build
	./bin/migrate -config config.toml -dry-run

test:
	go test -v -short ./internal/... ./cmd/...

test-integration:
	go test -v -timeout 600s ./test/...

test-all:
	go test -v -timeout 600s ./test/...

clean:
	rm -rf bin/
	rm -rf api/

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-clean:
	docker-compose down -v

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f

docker-restart: docker-down docker-build docker-up

deps:
	go mod download
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
