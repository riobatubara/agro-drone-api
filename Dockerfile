# Dockerfile definition for Backend application service.

# Image will be build. This is the environment.
FROM golang:1.21-alpine as Build

# Copy all the files in repo to the inside the container at root location.
COPY . .

# Build binary at root location.
RUN GOPATH= go build -o /main cmd/main.go

####################################################################
# Actual image that will be using in production.
FROM alpine:latest

# Copy the binary from the build image to the production image.
COPY --from=Build /main .

# Port that application will be listening on.
EXPOSE 1323

# Command that will be executed when the container is started.
ENTRYPOINT ["./main"]