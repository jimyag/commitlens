.PHONY: build frontend test clean run-web run

build: frontend
	go build -o commitlens .

frontend:
	cd frontend && npm run build

test:
	go test ./...

run:
	./commitlens --config config.example.yaml

run-web:
	./commitlens --web --config config.example.yaml

clean:
	rm -f commitlens
	rm -rf frontend/dist
