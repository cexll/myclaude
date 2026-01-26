# Claude Code Multi-Agent Workflow System Makefile
# Quick deployment for BMAD and Requirements workflows

.PHONY: help install deploy-bmad deploy-requirements deploy-essentials deploy-advanced deploy-all deploy-commands deploy-agents clean test changelog

# Default target
help:
	@echo "Claude Code Multi-Agent Workflow - Quick Deployment"
	@echo ""
	@echo "Recommended installation: npx github:cexll/myclaude"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  install              - LEGACY: install all configurations (prefer npx github:cexll/myclaude)"
	@echo "  deploy-bmad          - Deploy BMAD workflow (bmad-pilot)"
	@echo "  deploy-requirements  - Deploy Requirements workflow (requirements-pilot)"
	@echo "  deploy-essentials    - Deploy Development Essentials workflow"
	@echo "  deploy-advanced      - Deploy Advanced AI Agents"
	@echo "  deploy-commands      - Deploy all slash commands"
	@echo "  deploy-agents        - Deploy all agent configurations"
	@echo "  deploy-all           - Deploy everything (commands + agents)"
	@echo "  test-bmad            - Test BMAD workflow with sample"
	@echo "  test-requirements    - Test Requirements workflow with sample"
	@echo "  changelog            - Update CHANGELOG.md using git-cliff"
	@echo "  clean                - Clean generated artifacts"
	@echo "  help                 - Show this help message"

# Configuration paths
CLAUDE_CONFIG_DIR = ~/.claude
SPECS_DIR = .claude/specs

# Workflow directories
BMAD_DIR = agents/bmad
REQUIREMENTS_DIR = agents/requirements
ESSENTIALS_DIR = agents/development-essentials
ADVANCED_DIR = advanced-ai-agents
OUTPUT_STYLES_DIR = output-styles

# Install all configurations
install: deploy-all
	@echo "⚠️  LEGACY PATH: make install will be removed in future versions."
	@echo "    Prefer: npx github:cexll/myclaude"
	@echo "✅ Installation complete!"

# Deploy BMAD workflow
deploy-bmad:
	@echo "🚀 Deploying BMAD workflow..."
	@mkdir -p $(CLAUDE_CONFIG_DIR)/commands
	@mkdir -p $(CLAUDE_CONFIG_DIR)/agents
	@mkdir -p $(CLAUDE_CONFIG_DIR)/output-styles
	@cp $(BMAD_DIR)/commands/bmad-pilot.md $(CLAUDE_CONFIG_DIR)/commands/
	@cp $(BMAD_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@cp $(OUTPUT_STYLES_DIR)/bmad.md $(CLAUDE_CONFIG_DIR)/output-styles/ 2>/dev/null || true
	@echo "✅ BMAD workflow deployed successfully!"
	@echo "   Usage: /bmad-pilot \"your feature description\""

# Deploy Requirements workflow
deploy-requirements:
	@echo "🚀 Deploying Requirements workflow..."
	@mkdir -p $(CLAUDE_CONFIG_DIR)/commands
	@mkdir -p $(CLAUDE_CONFIG_DIR)/agents
	@cp $(REQUIREMENTS_DIR)/commands/requirements-pilot.md $(CLAUDE_CONFIG_DIR)/commands/
	@cp $(REQUIREMENTS_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@echo "✅ Requirements workflow deployed successfully!"
	@echo "   Usage: /requirements-pilot \"your feature description\""

# Deploy Development Essentials workflow
deploy-essentials:
	@echo "🚀 Deploying Development Essentials workflow..."
	@mkdir -p $(CLAUDE_CONFIG_DIR)/commands
	@mkdir -p $(CLAUDE_CONFIG_DIR)/agents
	@cp $(ESSENTIALS_DIR)/commands/*.md $(CLAUDE_CONFIG_DIR)/commands/
	@cp $(ESSENTIALS_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@echo "✅ Development Essentials deployed successfully!"
	@echo "   Available commands: /ask, /code, /debug, /test, /review, /optimize, /bugfix, /refactor, /docs, /think"

# Deploy Advanced AI Agents
deploy-advanced:
	@echo "🚀 Deploying Advanced AI Agents..."
	@mkdir -p $(CLAUDE_CONFIG_DIR)/agents
	@cp $(ADVANCED_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@echo "✅ Advanced AI Agents deployed successfully!"

# Deploy all commands
deploy-commands:
	@echo "📦 Deploying all slash commands..."
	@mkdir -p $(CLAUDE_CONFIG_DIR)/commands
	@cp $(BMAD_DIR)/commands/*.md $(CLAUDE_CONFIG_DIR)/commands/
	@cp $(REQUIREMENTS_DIR)/commands/*.md $(CLAUDE_CONFIG_DIR)/commands/
	@cp $(ESSENTIALS_DIR)/commands/*.md $(CLAUDE_CONFIG_DIR)/commands/
	@echo "✅ All commands deployed!"
	@echo "   Available commands:"
	@echo "   - /bmad-pilot (Full agile workflow)"
	@echo "   - /requirements-pilot (Requirements-driven)"
	@echo "   - /ask, /code, /debug, /test, /review"
	@echo "   - /optimize, /bugfix, /refactor, /docs, /think"

# Deploy all agents
deploy-agents:
	@echo "🤖 Deploying all agents..."
	@mkdir -p $(CLAUDE_CONFIG_DIR)/agents
	@cp $(BMAD_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@cp $(REQUIREMENTS_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@cp $(ESSENTIALS_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@cp $(ADVANCED_DIR)/agents/*.md $(CLAUDE_CONFIG_DIR)/agents/
	@echo "✅ All agents deployed!"

# Deploy everything
deploy-all: deploy-commands deploy-agents
	@mkdir -p $(CLAUDE_CONFIG_DIR)/output-styles
	@cp $(OUTPUT_STYLES_DIR)/*.md $(CLAUDE_CONFIG_DIR)/output-styles/ 2>/dev/null || true
	@echo "✨ Full deployment complete!"
	@echo ""
	@echo "Quick Start:"
	@echo "  BMAD:         /bmad-pilot \"build user authentication\""
	@echo "  Requirements: /requirements-pilot \"implement JWT auth\""
	@echo "  Manual:       /ask → /code → /test → /review"

# Test BMAD workflow
test-bmad:
	@echo "🧪 Testing BMAD workflow..."
	@echo "Run in Claude Code:"
	@echo '/bmad-pilot "Simple todo list with add/delete functions"'

# Test Requirements workflow
test-requirements:
	@echo "🧪 Testing Requirements workflow..."
	@echo "Run in Claude Code:"
	@echo '/requirements-pilot "Basic CRUD API for products"'

# Clean generated artifacts
clean:
	@echo "🧹 Cleaning artifacts..."
	@rm -rf $(SPECS_DIR)
	@echo "✅ Cleaned!"

# Quick deployment shortcuts
bmad: deploy-bmad
requirements: deploy-requirements
essentials: deploy-essentials
advanced: deploy-advanced
all: deploy-all

# Version info
version:
	@echo "Claude Code Multi-Agent Workflow System v3.1"
	@echo "BMAD + Requirements-Driven Development"

# Update CHANGELOG.md using git-cliff
changelog:
	@echo "📝 Updating CHANGELOG.md with git-cliff..."
	@if ! command -v git-cliff > /dev/null 2>&1; then \
		echo "❌ git-cliff not found. Installing via Homebrew..."; \
		brew install git-cliff; \
	fi
	@git-cliff -o CHANGELOG.md
	@echo "✅ CHANGELOG.md updated successfully!"
	@echo ""
	@echo "Preview the changes:"
	@echo "  git diff CHANGELOG.md"
