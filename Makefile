
migrate:
	migrate -source file://./pkg/storage/migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up

migrate-down:
	migrate -source file://./pkg/storage/migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" down