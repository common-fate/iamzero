
migrate:
	migrate -source file://./pkg/storage/migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up

migrate-down:
	migrate -source file://./pkg/storage/migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" down

test-postgres:
	go test ./pkg/storage/postgres_integration_tests/... -tags=postgres