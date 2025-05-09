.PHONY: clean test

clean:
	rm -rf sloghandler dist/

test:
	go test -v ./...
