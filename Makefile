BINARY := petkit
MODULE := github.com/vukyn/petkit

.PHONY: build install check fmt

## build — compile the binary into bin/
build:
	go build -o bin/$(BINARY) ./cmd/petkit

## install — put petkit on PATH at the version git says it is.
## `go install module@version` is what makes `petkit version` report a tag:
## the toolchain stamps the module version into the build info, which is the
## only place the version comes from. Installing from the working tree instead
## gives a binary that honestly reports (devel).
install:
	go install ./cmd/petkit

## check — the gate. Formatting first, then vet, then the tests.
##
## ⚠️ `gofmt -l .` prints the names of unformatted files and exits 0, so calling
## it directly passes a check it was meant to fail. The output is captured and
## the recipe fails when it is non-empty. This exact defect shipped in another
## repository here and went unnoticed for months.
check:
	@unformatted="$$(gofmt -l . 2>/dev/null)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt: these files are not formatted — run make fmt:"; \
		echo "$$unformatted" | sed 's/^/  /'; \
		exit 1; \
	fi
	go vet ./...
	go test ./... -count=1

## fmt — format every file in place
fmt:
	gofmt -w .
