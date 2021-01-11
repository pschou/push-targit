PROG_NAME := "push-targit"
IMAGE_NAME := "pschou/push-targit"
VERSION := "0.2"


build:
	CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}'" -o ${PROG_NAME} main.go

docker: build
	docker build -f Dockerfile --tag ${IMAGE_NAME}:${VERSION} .
