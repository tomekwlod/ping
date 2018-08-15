FROM alpine:3.8
LABEL maintainer="Tomasz Wlodarczyk"

COPY configs/               /app/configs/
COPY ./cmd/ping/ping        /app/ping
COPY ./cmd/server/server    /app/server

RUN chmod +x /app/ping \
    && chmod +x /app/server

ENV CONFIG_PATH /app/configs/
ENV PING_PORT 8080
EXPOSE 8080

ENTRYPOINT /app/server


# jenkins to get the code
# release it to a server
# build the docker containers (using multi stage builds)

# Example of how to use docker + postgres + go in prod mode
# https://medium.com/@McMenemy/godorp-docker-compose-for-development-and-production-e37fe0a58d61

# Local docker + golang + db --nice article for the dev
# https://medium.com/firehydrant-io/docker-compose-go-c2bb7bed0bab

# Deploying go + docker ==> uses docker hub
# https://medium.com/travis-on-docker/how-to-dockerize-your-go-golang-app-542af15c27a2

# Building go app with docker
# https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/

# Very usefull multi-stage build
# https://flaviocopes.com/golang-docker/#multi-stage-builds



# docker pull $1/ma-prod:latest
# if docker stop ma-app; then docker rm ma-app; fi
# docker run -d -p 8080:8080 --name ma-app $1/ma-prod
# if docker rmi $(docker images --filter "dangling=true" -q --no-trunc); then :; fi

# Building it (----> implement a name)


# AN EXAMPLE FROM DOCKER.COM https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#exclude-with-dockerignore
# FROM golang:1.9.2-alpine3.6 AS build

# # Install tools required for project
# # Run `docker build --no-cache .` to update dependencies
# RUN apk add --no-cache git
# RUN go get github.com/golang/dep/cmd/dep

# # List project dependencies with Gopkg.toml and Gopkg.lock
# # These layers are only re-built when Gopkg files are updated
# COPY Gopkg.lock Gopkg.toml /go/src/project/
# WORKDIR /go/src/project/
# # Install library dependencies
# RUN dep ensure -vendor-only

# # Copy the entire project and build it
# # This layer is rebuilt when a file changes in the project directory
# COPY . /go/src/project/
# RUN go build -o /bin/project

# # This results in a single layer image
# FROM scratch
# COPY --from=build /bin/project /bin/project
# ENTRYPOINT ["/bin/project"]
# CMD ["--help"]