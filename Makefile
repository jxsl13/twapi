


build: build-macos build-linux build-windows

build-macos:
	GOOS=darwin go build ./...

build-linux:
	GOOS=linux go build ./...

build-windows:
	GOOS=windows go build ./...