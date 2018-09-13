# First step - just building the go app
FROM golang:1.9.7 as builder
ENV WORKDIR /go/src/app
WORKDIR ${WORKDIR}
COPY . .
RUN export \
    # && go get -d -v ./... \
    && go get -u github.com/golang/dep/cmd/dep \
    && dep init && dep ensure \
    && cd ${WORKDIR}/cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -o server -a -installsuffix cgo server.go handler.go router.go

# Second step - copying the files and running the exec
FROM alpine:3.8
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /go/src/app/cmd/server/server .
# below won't work for CI because we won't have the .gitignored yml files
# we have to generate/copy them before this action (TODO)
COPY --from=builder /go/src/app/configs/parameters.yml configs/parameters.yml

# ENV CONFIG_PATH /root/configs
# ENV PING_PORT 8080
# EXPOSE 8080

ENTRYPOINT ["./server"]





# env variables via ARGs
# docker build --no-cache --build-arg CONFIG_PATH="path/here" -t godocker -f deployments/Dockerfile .
# ARG CONFIG_PATH
# COPY $CONFIG_PATH/db.yml configs/db.yml
# COPY $CONFIG_PATH/smtp.yml configs/smtp.yml
# https://stackoverflow.com/questions/19537645/get-environment-variable-value-in-dockerfile

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