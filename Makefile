BIN := build/brc

.PHONY: build run small clean

build:
	go build -o $(BIN) .

run: build
	./$(BIN) 1brc/measurements.txt > /dev/null

small: build
	./$(BIN) weather_stations_small.txt

clean:
	rm -rf build
