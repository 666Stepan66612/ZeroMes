.PHONY: all local

all:
	@docker compose down
	@cd frontend && npm install
	@cd frontend && npm run build
	@docker compose up -d --build

local:
	@docker compose down
	@docker compose up -d --build
	@cd frontend && npm install && npm run dev
