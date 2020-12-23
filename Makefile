PROG_NAME := "post-targit"
IMAGE_NAME := "pschou/post-targit"
VERSION := "0.1"


build:
	CGO_ENABLED=0 go build -o ${PROG_NAME} main.go

docker: build
	docker build -f Dockerfile --tag ${IMAGE_NAME}:${VERSION} .
