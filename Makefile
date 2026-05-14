.PHONY: all local

all:
	@cd src/frontend && npm install
	@cd src/frontend && npm run build
	@cd src && docker compose down
	@cd src && docker compose up -d --build

local:
	@cd src && docker compose down
	@cd src && docker compose up -d --build
	@cd src/frontend && npm install && npm run dev
