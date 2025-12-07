web:
	cd web && npm run dev

backend:
	cd backend && go run main.go

.PHONY: web backend
