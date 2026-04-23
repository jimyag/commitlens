.PHONY: build frontend test clean run-web run

frontend:
	cd frontend && npm run build

build: frontend
	go build -o commitlens .

test:
	go test ./...

run:
	./commitlens --config config.example.yaml

run-web:
	./commitlens --web --config config.example.yaml

clean:
	rm -f commitlens
	rm -rf frontend/dist
