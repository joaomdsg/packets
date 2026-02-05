# agntpr

A sandboxed, semi-autonomous coding agent controlled from GitHub ―
forks, plans, implements, and submits pull requests.

## Features

- **GitHub-Driven Workflow**: Assign issues by mentioning the agent,
  receive implementation plans for approval
- **Fork-Based Safety**: All work happens in a fork with PR-based
  review before merging
- **Test-Driven Development**: Enforces RED-GREEN-REFACTOR cycle for
  quality code
- **State Machine**: Tracks issues through planning → implementation →
  PR review → done
- **Maintainer Control**: Plan approval required, can skip planning,
  maintains authority throughout
- **Intent Classification**: Automatically determines if comments are
  approvals, revisions, questions, or clarifications

## Quick Start

### Prerequisites

- Docker and docker-compose
- GitHub Personal Access Token with `repo` scope
- Anthropic API key for Claude

### Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/agntpr.git
cd agntpr
```

2. Create `.env` file:
```bash
GITHUB_TOKEN=ghp_your_token_here
CLAUDE_API_KEY=sk-ant-your_key_here
TARGET_REPO=owner/repo
CLAUDE_MODEL=sonnet
POLL_INTERVAL=60
```

3. Run with docker-compose:
```bash
docker-compose up -d
```

4. View logs:
```bash
docker-compose logs -f agntpr
```

### Usage

1. **Create an issue** in your target repository
2. **Mention the agent** in a comment (e.g.,
   `@ai-r-sentry please implement this`)
3. **Review the plan** posted by the agent
4. **Approve with** `@ai-r-sentry approve` or request changes with
   `@ai-r-sentry revise: <feedback>`
5. **Review the PR** created by the agent
6. **Merge** when satisfied

Skip planning for simple issues: `@ai-r-sentry just implement this directly`

## How It Works

```
┌─────────────┐
│ GitHub Issue│ ← Agent watches for mentions
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Create Fork  │ ← Isolated work environment
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Create Plan  │ ← 4-section implementation plan
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Wait Approval│ ← Maintainer reviews plan
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Implement   │ ← TDD: RED → GREEN → REFACTOR
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Create PR   │ ← PR from fork to upstream
└──────┬──────┘
       │
       ▼
┌─────────────┐
│Review Cycle │ ← Address feedback, iterate
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Merged    │
└─────────────┘
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GITHUB_TOKEN` | GitHub PAT with repo scope | Required |
| `CLAUDE_API_KEY` | Anthropic API key | Required |
| `TARGET_REPO` | Repository to watch (owner/repo) | Required |
| `CLAUDE_MODEL` | Claude model (sonnet/haiku/opus) | `sonnet` |
| `POLL_INTERVAL` | Polling interval in seconds | `60` |
| `RESET_DB` | Reset database on startup | `false` |
| `DEBUG` | Enable debug logging | `false` |
| `DATABASE_PATH` | SQLite database path | `/data/agntpr.db` |
| `WORK_DIR` | Working directory for forks | `/work` |

## Local Development

Build and run locally without Docker:

```bash
# Install dependencies
go mod download

# Build
go build -o agntpr ./cmd/agntpr

# Run tests
go test ./...

# Run with environment variables
export GITHUB_TOKEN=ghp_...
export CLAUDE_API_KEY=sk-ant-...
export TARGET_REPO=owner/repo
./agntpr
```

GitHub CLI must be authenticated:
```bash
gh auth login
```

## Architecture

See [CLAUDE.md](CLAUDE.md) for detailed architecture documentation
including:

- State machine transitions
- Component interactions (Orchestrator, Watcher, ForkManager, Agent)
- Agent prompt design
- Database schema

## Security

- Agent executes in isolated sandbox with limited permissions
- Fork-based workflow prevents direct changes to target repository
- All changes require maintainer approval via PR
- No credentials exposed to agent execution environment
- Context files (`.agntpr-context.md`) never committed to git

## License

MIT
