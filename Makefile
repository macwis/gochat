build:
	go build -o bin/gochat -v .

clean:
	go clean
	rm -rf bin
