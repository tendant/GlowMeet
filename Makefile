web:
	cd web && npm run dev

backend:
	cd backend && go run main.go

backend-watch:
	cd backend && arelo -p . -- go run main.go

.PHONY: web backend backend-watch
