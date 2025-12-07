web:
	cd web && npm run dev

backend:
	cd backend && go run main.go

backend-watch:
	cd backend && arelo -t . -p '**/*.go' -- go run .

test:
	cd backend && go test -v ./...

.PHONY: web backend backend-watch test
