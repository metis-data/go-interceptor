build-e2e:
	GOOS=linux go build -o e2e/web-gorilla-gorm/web-gorilla-gorm e2e/web-gorilla-gorm/main.go
	GOOS=linux go build -o e2e/web/web e2e/web/main.go
	GOOS=linux go build -o e2e/web-gorilla-sqlz/web-gorilla-sqlz e2e/web-gorilla-sqlz/main.go

run-e2e: build-e2e
	docker-compose -f e2e/docker-compose.yml down -v --remove-orphans
	docker-compose -f e2e/docker-compose.yml up --build --abort-on-container-exit --remove-orphans

unittest:
	go test ./...

.PHONY: build-e2e run-e2e unittest