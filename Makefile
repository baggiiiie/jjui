test:
	go test ./...

build: test
	go build ./cmd/jjui
