
migrate:
	migrate -source file://./migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up

migrate-down:
	migrate -source file://./migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" down