# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when
working with code in this repository.

## Project Overview

agntpr is an autonomous Claude Code agent orchestrator that safely
monitors GitHub repositories for issues and PRs, then addresses them
through a structured TDD workflow with maintainer collaboration.

## Core Workflow

1. **Fork & Watch** - Create/maintain a fork of a target repo, watch
   for issues assigned to the authenticated GitHub user
2. **Plan** - Sync fork with upstream, read issue, respond with a plan
   including test descriptions, wait for maintainer approval
3. **Refine** - Iterate on plan based on maintainer feedback until
   approved (maintainer can skip planning)
4. **Implement** - Sync fork with upstream, create branch from latest,
   use TDD to address the issue
5. **PR** - Create PR highlighting key changes for review
6. **Review Cycle** - Address maintainer comments on existing branch,
   iterate until merged/rejected
7. **PR Mentions** - Respond to `@ai-r-sentry` mentions in PR comments

**Note**: The fork is automatically synced with upstream (via
`git fetch` + `git reset --hard`) before planning and implementation
to ensure plans and branches are based on current code. Review
responses work on the existing branch without syncing.

## Architecture

### High-Level Flow

```
main.go → Orchestrator → State Machine → Agent (Claude Code)
   ↓           ↓              ↓              ↓
 Config    Database      Transitions    Fork/WorkDir
   ↓           ↓
Watcher  ForkManager
   ↓
GitHub API (via gh CLI)
```

### State Machine

Issues progress through states (`internal/state/machine.go`):

- **new** → **planning** → **plan_review** → **implementing** →
  **pr_created** → **pr_review** → **done/rejected**
- Can skip planning: **new** → **implementing**
- Terminal states: **done**, **rejected**, **errored**

Transitions triggered by events like `plan_approved`,
`implementation_complete`, `pr_merged`.

### Key Components

**Orchestrator** (`internal/orchestrator/orchestrator.go`):
- Main business logic coordinator
- Polls GitHub for mentioned issues/PRs
- Manages state transitions and workflow
- Coordinates between watcher, fork manager, and agent
- Handles error states and maintainer communication

**Watcher** (`internal/watcher/watcher.go`):
- GitHub API wrapper using `gh` CLI
- Fetches issues assigned to agent (via mentions)
- Fetches/filters comments on issues and PRs
- Posts comments and creates PRs

**ForkManager** (`internal/fork/manager.go`):
- Manages fork lifecycle and work directories
- Creates/syncs forks with upstream
- Syncs with upstream before every action (planning, implementation)
- Creates feature branches per issue
- Pushes changes back to fork

**Agent** (`internal/agent/`):
- Wraps Claude Code CLI invocations
- Builds specialized prompts for each phase
- `Invoker` handles prompt construction
- `ClaudeRunner` executes `claude` CLI with flags
- Five agent modes: Plan, Implement, RespondToReview,
  SummarizeChanges, EvaluateIntent, AnswerQuestion

**Context** (`internal/context/context.go`):
- Writes `.agntpr-context.md` in work directories
- Contains issue body, comments, PR feedback, plan history
- Provides full context to Claude Code agent
- **Never committed** to Git

**Database** (`internal/db/db.go`):
- SQLite persistence layer
- Tracks issues, plans, PRs, comments
- Schema defined in `models.go`
- State and workflow metadata

### Agent Execution

Agent phases correspond to state machine transitions:

1. **Planning** - Analyzes issue, proposes implementation plan with
   tests
2. **Implementation** - Uses TDD to implement approved plan
3. **Review Response** - Addresses PR review comments
4. **Summarization** - Creates PR description from git diff
5. **Intent Evaluation** - Parses maintainer comments as structured
   JSON to determine next action
6. **Question Answering** - Responds to questions without making code
   changes

Each phase runs Claude Code CLI in the work directory with a
specialized prompt from `agent/invoker.go`.

### Agent Prompt Design

All agent prompts follow these principles:

**Structured Formats**: Each prompt provides clear sections and
expected output format
- Planning: 4-section structure (Analysis, Implementation, Tests,
  Risks), 300-500 words
- Implementation: RED-GREEN-REFACTOR-REPEAT cycle
- Intent Evaluation: 5 categories with single-line JSON output

**TDD Enforcement**: Strict test-first methodology across all phases
- Write failing test first (RED)
- Minimal code to pass (GREEN)
- Refactor for clarity
- Commit when logical units complete

**Scope Boundaries**: Clear guidance on what the agent should and shouldn't do
- Stay within approved plan
- Maintainer has final authority
- Document interpretations in commit messages
- 30-minute execution timeout

**Context Awareness**: All prompts reference `.agntpr-context.md` file
containing issue details, comments, PR feedback, and plan history.
This file is never committed to git.

Prompts are optimized for token efficiency (~1,700 tokens total) while
maintaining clarity and effectiveness.

## Build & Development Commands

### Local Development

```bash
# Build the binary
go build -o agntpr ./cmd/agntpr

# Run tests
go test ./...

# Run specific test
go test -v ./internal/state -run TestTransition

# Run with coverage
go test -cover ./...
```

### Docker Development

```bash
# Build and run with docker-compose
docker-compose up --build

# Run in detached mode
docker-compose up -d

# View logs
docker-compose logs -f agntpr

# Stop and clean up
docker-compose down -v
```

### Environment Configuration

Copy `.env.example` to `.env` and configure:

- `GITHUB_TOKEN` - GitHub PAT with repo scope
- `CLAUDE_API_KEY` - Anthropic API key (set as `ANTHROPIC_API_KEY` in
  docker-compose)
- `TARGET_REPO` - Repository to watch (format: `owner/repo`)
- `CLAUDE_MODEL` - Model name (default: `sonnet`, also `haiku` or
  `opus`)
- `POLL_INTERVAL` - Polling interval in seconds (default: `60`)
- `RESET_DB` - Reset database on startup (default: `false`)
- `DEBUG` - Enable debug logging (default: `false`)

### GitHub Authentication

The agent uses GitHub CLI (`gh`) for repository operations. In Docker,
the `GITHUB_TOKEN` env var is used. Locally, authenticate:

```bash
gh auth login
```

### Database

Uses SQLite (default: `/data/agntpr.db` in Docker, configurable via
`DATABASE_PATH`). Reset with `RESET_DB=true`.

## Security Considerations

- Agent runs in isolated sandbox with limited permissions
- Fork-based workflow prevents direct changes to origin
- All code changes require maintainer approval via PR
- No credential exposure to agent execution environment
