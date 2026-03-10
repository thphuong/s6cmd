BINARY := s6cmd

.PHONY: build test e2e clean install

build:
	go build -o $(BINARY) .

test:
	go test ./...

e2e: build
	./e2e_test.sh --profile $(or $(PROFILE),local)

clean:
	rm -f $(BINARY)

install:
	go install .
