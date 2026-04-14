.PHONY: all

all:
	@docker compose down
	@cd frontend && npm run build
	@docker compose up -d --build
