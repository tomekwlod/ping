# build:
# 	cd cmd/ping && GOOS=linux GOARCH=amd64 go build -v -o ping
# 	cd cmd/server && GOOS=linux GOARCH=amd64 go build -v -o server
# 	make dockerbuild
# run:
# 	docker run -i -p 8080:8080 goping:1.0.0

dockerbuild:
	$(if $(shell docker images -a | grep goping),@echo "Docker is ready",docker build --no-cache -t goping:1.0.0 -f deployments/Dockerfile .)
