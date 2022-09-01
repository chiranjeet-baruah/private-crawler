install:
	go mod vendor 

test:
	go vet ./...

binary-build:
	echo "compiling with `go version`"
	CGO_ENABLED=0 GOOS=linux go build -o ./go-crawler
	echo "built ./go-crawler"

build: test binary-build
