all: zeus-cli check

zeus-cli:
	go build

check:
	go test
	go vet

clean:
	rm -f zeus-cli

.PHONY: check clean zeus-cli
