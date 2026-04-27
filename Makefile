BINARY := vor
ALIAS := cs
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
INSTALL_DIR := /usr/local/bin
GOMOBILE := $(HOME)/go/bin/gomobile

.PHONY: build install install-completions uninstall test test-cscore test-mobile fuzz-cscore mobile-ios mobile-clean lint audit-see-also audit-see-also-strict fmt clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/$(BINARY)/

install: build
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	ln -sf $(INSTALL_DIR)/$(BINARY) $(INSTALL_DIR)/$(ALIAS)
	@echo ""
	@echo "Installed:"
	@echo "  $(INSTALL_DIR)/$(BINARY)        (binary)"
	@echo "  $(INSTALL_DIR)/$(ALIAS) -> $(BINARY)  (backward-compat alias)"
	@$(MAKE) -s install-completions

install-completions: build
	@echo ""
	@echo "Installing shell tab-completions..."
	@# bash — try Homebrew (macOS), then system (/etc), then user-level
	@for dir in /opt/homebrew/etc/bash_completion.d /usr/local/etc/bash_completion.d /etc/bash_completion.d; do \
		if [ -d "$$dir" ] && [ -w "$$dir" ]; then \
			./$(BINARY) --completions bash > "$$dir/$(BINARY)" 2>/dev/null && echo "  ✓ bash: $$dir/$(BINARY)"; \
			cp -f "$$dir/$(BINARY)" "$$dir/$(ALIAS)" 2>/dev/null && \
				sed -i.bak -e 's/_$(BINARY)/_$(ALIAS)/g' -e 's/$(BINARY) --completions-list/$(ALIAS) --completions-list/g' -e 's/complete -F _$(ALIAS) $(BINARY)/complete -F _$(ALIAS) $(ALIAS)/' "$$dir/$(ALIAS)" 2>/dev/null && rm -f "$$dir/$(ALIAS).bak" && echo "  ✓ bash: $$dir/$(ALIAS) (alias)"; \
			break; \
		fi; \
	done
	@# zsh — Homebrew on macOS most common; system as fallback
	@for dir in /opt/homebrew/share/zsh/site-functions /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions; do \
		if [ -d "$$dir" ] && [ -w "$$dir" ]; then \
			./$(BINARY) --completions zsh > "$$dir/_$(BINARY)" 2>/dev/null && echo "  ✓ zsh:  $$dir/_$(BINARY)"; \
			ARGV0="$(ALIAS) " ./$(BINARY) --completions zsh 2>/dev/null > "$$dir/_$(ALIAS)" && echo "  ✓ zsh:  $$dir/_$(ALIAS) (alias)" || true; \
			break; \
		fi; \
	done
	@# fish — per-user, always writable
	@mkdir -p "$$HOME/.config/fish/completions" 2>/dev/null && \
		./$(BINARY) --completions fish > "$$HOME/.config/fish/completions/$(BINARY).fish" 2>/dev/null && \
		echo "  ✓ fish: $$HOME/.config/fish/completions/$(BINARY).fish" && \
		sed 's/-c $(BINARY)/-c $(ALIAS)/g; s/$(BINARY) --completions-list/$(ALIAS) --completions-list/g' \
			"$$HOME/.config/fish/completions/$(BINARY).fish" \
			> "$$HOME/.config/fish/completions/$(ALIAS).fish" 2>/dev/null && \
		echo "  ✓ fish: $$HOME/.config/fish/completions/$(ALIAS).fish (alias)" || true
	@echo ""
	@echo "If completions don't activate, restart your shell or source your rc."
	@echo "Manual fallback:"
	@echo "  bash: eval \"\$$($(BINARY) --completions bash)\""
	@echo "  zsh:  eval \"\$$($(BINARY) --completions zsh)\""
	@echo "  fish: $(BINARY) --completions fish | source"
	@echo ""

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY) $(INSTALL_DIR)/$(ALIAS)

test:
	go test ./... -count=1 -race

test-cscore:
	go test ./pkg/cscore/... -count=1 -race -v

fuzz-cscore:
	go test ./pkg/cscore/ -fuzz=FuzzCalcEval -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzSubnetCalc -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzSearchJSON -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzRenderMarkdownToHTML -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzGetSheetJSON -fuzztime=30s

test-mobile:
	go test ./mobile/... -count=1 -race -v

mobile-ios:
	PATH="$(HOME)/go/bin:$(PATH)" $(GOMOBILE) bind -target=ios -ldflags="-s -w" -o mobile/Cscore.xcframework ./mobile/

mobile-clean:
	rm -rf mobile/Cscore.xcframework

lint:
	go vet ./...
	@$(MAKE) -s audit-see-also

audit-see-also:
	@scripts/audit-see-also.sh --allowlist=.ci/see-also-allowlist.txt

audit-see-also-strict:
	@scripts/audit-see-also.sh

fmt:
	gofmt -s -w .

clean:
	rm -f $(BINARY) $(ALIAS)
	rm -rf mobile/Cscore.xcframework
