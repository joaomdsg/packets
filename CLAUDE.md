# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when
working with code in this repository.

## Project Overview

agntpr is an autonomous Claude Code agent orchestrator that safely
monitors GitHub repositories for issues and PRs, then addresses them
through a structured TDD workflow with maintainer collaboration.

## Core Workflow

1. **Fork & Watch** - Create/maintain a fork of a target repo, watch
   for issues mentioning the agent (e.g., @agntpr)
2. **Clarify** - If issue is ambiguous or contains questions, agent
   asks for clarification and waits in "new" state for response
3. **Plan** - Sync fork with upstream, read issue, respond with full
   plan text including test descriptions, wait for maintainer approval
4. **Refine** - Iterate on plan based on maintainer feedback until
   approved (maintainer can skip planning)
5. **Implement** - Sync fork with upstream, create branch from latest,
   use TDD to address the issue
6. **PR** - Create PR highlighting key changes for review
7. **Review Cycle** - Address maintainer comments on existing branch,
   iterate until merged/rejected

**Note**: The fork is automatically synced with upstream (via
`git fetch` + `git reset --hard`) before planning and implementation
to ensure plans and branches are based on current code. Review
responses work on the existing branch without syncing.

**Question Handling**: When a user asks a question or the agent needs
clarification, the issue stays in "new" state. The agent waits for a
response mentioning it before proceeding to planning. This prevents
premature planning on unclear requirements.

## Architecture

### High-Level Flow

```text
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
- **Question/Clarification Flow**: Detects when agent needs
  clarification or user asks questions, posts response, and keeps
  issue in "new" state until maintainer responds with a mention

**Watcher** (`internal/watcher/watcher.go`):

- GitHub API wrapper using `gh` CLI
- Fetches issues mentioning agent (@username format)
- Fetches/filters comments on issues and PRs
- Posts comments and creates PRs
- Detects approval and revision commands in comments

**ForkManager** (`internal/fork/manager.go`):

- Manages fork lifecycle and work directories
- Creates/syncs forks with upstream
- Syncs with upstream before every action (planning, implementation)
- Creates feature branches per issue
- Pushes changes back to fork

**Agent** (`internal/agent/`):

- Wraps AI backend CLI invocations (OpenCode or Claude Code)
- Builds specialized prompts for each phase
- `Invoker` handles prompt construction
- `ClaudeRunner` executes `claude` CLI with flags
- `OpenCodeRunner` executes `opencode` CLI with flags
- Six agent modes: Plan, Implement, RespondToReview,
  SummarizeChanges, EvaluateIntent, AnswerQuestion
- **Plan Output**: Plans are returned as full text (not files) for
  direct posting to GitHub comments

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

**Scope Boundaries**: Clear guidance on what the agent should and
shouldn't do

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

Copy `example.env` to `.env` and configure:

- `GITHUB_TOKEN` - GitHub PAT with repo scope (for token auth)
- `GITHUB_APP_PRIVATE_KEY` - GitHub App private key (for app auth)
- `GITHUB_APP_INSTALLATION_ID` - Installation ID (for app auth)
- `TARGET_REPO` - Repository to watch (format: `owner/repo`)
- `AI_BACKEND` - AI backend (`opencode` or `claude`, default:
  `opencode`)
- `CLAUDE_API_KEY` - Anthropic API key (for Claude Code backend)
- `CLAUDE_MODEL` - Claude model (`sonnet`, `haiku`, or `opus`)
- `OPENCODE_MODEL` - OpenCode model (default: uses free Kimi K2.5)
- `POLL_INTERVAL` - Polling interval in seconds (default: `60`)
- `RESET_DB` - Reset database on startup (default: `false`)
- `DEBUG` - Enable debug logging (default: `false`)

### GitHub Authentication

The agent uses GitHub CLI (`gh`) for repository operations and supports
two authentication methods:

**Token Authentication** (simpler, for personal use):

- Set `GITHUB_TOKEN` with a fine-grained PAT
- Token is used directly for all operations

**GitHub App Authentication** (recommended, for organizations):

- Create a GitHub App with repo permissions
- Set `GITHUB_APP_PRIVATE_KEY` and `GITHUB_APP_INSTALLATION_ID`
- Tokens are refreshed automatically (1-hour expiration)

Locally, you can also authenticate:

```bash
gh auth login
```

### Database

Uses SQLite (default: `/data/agntpr.db` in Docker, configurable via
`DATABASE_PATH`). Reset with `RESET_DB=true`.

## Markdown Standards

All markdown files in this repository must follow these rules:

**Line Length**: Maximum 80 characters per line
- Wrap long lines at natural break points
- Break list items with continuation indentation

**Blank Lines**:
- Around all headers
- Before and after code blocks
- Before all lists (MD032 compliance)
- After list items with continuation text

**Code Blocks**:
- Always specify language (bash, go, text, etc.)
- Use `text` for ASCII diagrams
- Indent within numbered lists (3 spaces)

**Lists**:
- Consistent `-` marker for unordered lists
- Proper indentation for nested items
- Blank line before first item when after text

**Headers**:
- ATX style with `#` only
- No trailing spaces
- Blank line before and after

**Tables**:
- Proper alignment with separator row
- Consistent column spacing

When modifying markdown files, verify compliance with:
```bash
awk 'length > 80' file.md  # Check line length
grep ' $' file.md           # Check trailing spaces
```

## Security Considerations

- Agent runs in isolated sandbox with limited permissions
- Fork-based workflow prevents direct changes to origin
- All code changes require maintainer approval via PR
- No credential exposure to agent execution environment
