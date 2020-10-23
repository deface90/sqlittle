.PHONY: all test ci bench format fuzz readme

all: format test

test:
	go test ./...

ci:
	go test -tags ci ./...

bench:
	go test -bench . github.com/deface90/sqlittle/db

format:
	go fmt

fuzz:
	go get -v github.com/dvyukov/go-fuzz/...

	rm -f sqlittle-fuzz.zip
	go-fuzz-build github.com/deface90/sqlittle/db
	mkdir -p workdir
	cp -r corpus workdir
	go-fuzz -bin=sqlittle-fuzz.zip -workdir=workdir
