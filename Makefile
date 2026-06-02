.PHONY: build test vet fmt clean run

build:
	go build -o blockroll-server ./cmd/blockroll-server
	go build -o blockroll-cli ./cmd/blockroll-cli

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f blockroll-server blockroll-server.exe blockroll-cli blockroll-cli.exe
	rm -f coverage.out coverage.html

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run: build
	./blockroll-server
