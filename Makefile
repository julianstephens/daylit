
.PHONY: lint-*

lint-cli:
	@echo "Linting CLI source files..."
	@cd daylit-cli && make lint

lint-tray:
	@echo "Linting Tray source files..."
	@cd daylit-tray && npm run format && npm run lint
