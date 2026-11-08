BIN := build/brc

.PHONY: build run small test profile clean

build:
	go build -o $(BIN) .

run: build
	./$(BIN) 1brc/measurements.txt > /dev/null

small: build
	./$(BIN) weather_stations_small.txt

clean:
	rm -rf build

test: build
	go test .
	./test.sh

profile: build
	./$(BIN) -cpuprofile build/cpu.prof 1brc/measurements.txt > /dev/null
	go tool pprof -top build/brc build/cpu.prof | head -25
