.PHONY: all

all:
	@cd frontend && npm run build
	@docker-compose up -d --build
