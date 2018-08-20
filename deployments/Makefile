ifeq ("$(wildcard .env)","")
$(error Dotenv (.env) file doesn't exist. It is needed here!)
endif

stop:
	docker-compose stop

start:
	docker-compose build
	docker-compose up -d --build
rm:
	make stop
	docker-compose rm -f

rebuild:
	make rm
	make start