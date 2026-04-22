.PHONY: all

all:
	@git pull
	@git commit -m "push"
	@docker compose down
	@cd frontend && npm install
	@cd frontend && npm run build
	@docker compose up -d --build
