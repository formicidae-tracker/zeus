all: zeus-calibrator check

.PHONY: check clean

zeus-calibrator: *.go
	go generate
	go build

check:
	go test
	go vet

clean:
	rm -f zeus-calibrator
