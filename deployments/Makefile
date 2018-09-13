ifeq ("$(wildcard .env)","")
$(error Dotenv (.env) file doesn't exist. It is needed here!)
endif

stop:
	docker-compose stop

build:
	docker-compose build

start:
	docker-compose up -d --build

destroy:
	make stop
	docker-compose rm -f

rebuild:
	make destroy
	make build
	make start