migrate-install:
	brew install golang-migrate
migrate-up:
	migrate -database ${POSTGRESQL_URL} -path db/migration up
migrate-down:
	migrate -database ${POSTGRESQL_URL} -path db/migration down 1
migrate-create:
	migrate create -ext sql -dir db/migration -seq ${NAME}

sqlc-install:
	brew install sqlc
sqlc-init:
	sqlc init
sqlc-generate:
	sqlc generate
