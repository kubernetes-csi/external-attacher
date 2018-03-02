all: build

PROG := mock

$(PROG): *.go ./provider/*.go ./service/*.go
	@go install .
	go build -o "$@" .

build: $(PROG)

clean:
	go clean -i -v .
	rm -f $(PROG)

clobber: clean

.PHONY: clean clobber
