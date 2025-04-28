## ========== Local Development 指令 ==========
.PHONY: swag test tidy run

test:
	go test ./internal/...

tidy:
	go mod tidy

local-run:
	go run cmd/server/main.go

local-migrate:
	go run cmd/server/main.go -migrate


local-seed:
	go run cmd/server/main.go -seed

## ========== Docker Compose 指令 ==========
.PHONY: up down clean restart migrate seed rebuild-app

up:
	docker-compose up --build

down:
	docker-compose down

clean:
	docker-compose down -v

restart:
	make down && make up

rebuild-app:
	docker-compose build app
	docker-compose up -d app

migrate:
	docker-compose exec app ./hr-app -migrate

seed:
	docker-compose exec app ./hr-app -seed


# up:
# 	docker-compose up --build

# down:
# 	docker-compose down
	
# clean:
# 	docker-compose down -v

# restart:
# 	make down && make up

# rebuild-app:
# 	docker-compose build app
# 	docker-compose up -d app

# migrate:
# 	docker-compose exec app ./hr-app -migrate

# seed:
# 	docker-compose exec app ./hr-app -seed