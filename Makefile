.PHONY: build genkey host client clean docker-up docker-down

build:
	@echo "Building T2C daemon..."
	@go build -o t2c cmd/t2c/main.go
	@echo "Build complete! Executable 't2c' created."

genkey: build
	@echo "Generating new identity..."
	@./t2c genkey

host: build
	@echo "Starting T2C Host Node on port 4001, exposing local port 8000..."
	@mkdir -p data
	@if [ ! -f data/serveur.txt ]; then ./t2c genkey > data/serveur.txt; fi
	@KEY=$$(cat data/serveur.txt | grep "Private Key:" | awk '{print $$3}') && \
	./t2c start --key $$KEY --expose 8000 --port 4001

client: build
	@echo "Starting T2C Client Proxy on port 8080..."
	@mkdir -p data
	@if [ ! -f data/client.txt ]; then ./t2c genkey > data/client.txt; fi
	@KEY=$$(cat data/client.txt | grep "Private Key:" | awk '{print $$3}') && \
	./t2c start --key $$KEY --proxy 8080 --port 4002

docker-up:
	@echo "Starting Docker simulation..."
	docker-compose up --build -d

docker-down:
	@echo "Stopping Docker simulation..."
	docker-compose down

clean:
	@echo "Cleaning up generated files..."
	rm -f t2c
	rm -rf data
