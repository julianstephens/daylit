
.PHONY: lint* test*

lint-cli:
	@echo "Linting CLI source files..."
	@cd daylit-cli && make lint

lint-tray:
	@echo "Linting Tray source files..."
	@cd daylit-tray && npm run format && npm run lint && npm run build
	@cd daylit-tray/src-tauri && cargo clippy --workspace --all-targets --all-features -- -D warnings && cargo fmt --all -- --check

lint: lint-cli lint-tray
	@echo "All linting tasks completed."

test-cli:
	@echo "Running CLI tests..."
	@cd daylit-cli && make test

test-tray:
	@echo "Running Tray tests..."
	@cd daylit-tray && npm run build
	@cd daylit-tray/src-tauri && cargo test

test: test-cli test-tray
	@echo "All tests completed."