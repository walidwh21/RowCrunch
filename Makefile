.PHONY: build test vet data run bench clean

ROWS    ?= 1000000000
INPUT   ?= data/measurements.csv
WORKERS ?= $(shell nproc 2>/dev/null || echo 4)

build:
	go build -o bin/crunch ./cmd/crunch
	go build -o bin/datagen ./cmd/datagen

test:
	go test ./...

vet:
	go vet ./...

# Generate the benchmark input (about 16 GB for one billion rows).
data: build
	mkdir -p data
	./bin/datagen -rows $(ROWS) -out $(INPUT)

run: build
	./bin/crunch -input $(INPUT) -mode parallel -workers $(WORKERS)

# Compare both strategies on the same input; appends to data/runs.log.
bench: build
	./scripts/bench.sh $(INPUT)

clean:
	rm -rf bin
