# 🤖 AgntPR

An autonomous AI agent that watches your GitHub issues and ships PRs.
No IDE required. No copy-paste. Just mention it and let it work.

## Why AgntPR?

Traditional AI coding assistants require constant interaction—sitting
in your IDE, waiting for commands, needing prompts copy-pasted, context
manually provided. AgntPR is different:

- **Lives in GitHub** - Mention the agent on any issue, it handles the
  rest
- **Autonomous workflow** - Forks, plans, implements, and opens PRs
  without babysitting
- **Intelligent clarification** - Asks questions when requirements are
  unclear, waits for answers
- **Safety first** - Fork-based isolation, plan approval required,
  maintainer stays in control
- **TDD enforced** - RED-GREEN-REFACTOR cycle baked into every
  implementation
- **Transparent process** - Full plans posted as comments, not hidden
  in files

## How It Works

1. **Mention** - Tag the agent on any issue (e.g., `@agntpr`)
2. **Clarify** - If the issue is unclear or asks a question, the agent
   responds and waits for your answer
3. **Plan** - Agent forks your repo, analyzes the issue, posts a
   detailed implementation plan
4. **Approve** - Review the plan, request changes, or approve to
   proceed
5. **Implement** - Agent uses TDD to implement the plan, commits with
   clear messages
6. **Review** - Agent opens a PR from its fork, you review and merge
   when ready

Skip planning for simple changes by saying "just implement this
directly" in your mention.

## Quick Start

### 1. Create a GitHub Account for the Agent

Create a dedicated GitHub account (e.g., `your-org-bot`). Never use
your personal account.

### 2. Set Up Authentication

**Option A: Personal Access Token** (simpler, good for testing)

- Go to GitHub Settings → Developer settings → Personal access tokens
  → Fine-grained tokens
- Generate token with `Contents: Read and write` permission
- Scope to only repositories the agent should access

**Option B: GitHub App** (recommended for organizations)

- Create a GitHub App with `Contents: Read and write` permission
- Install the app on your target repositories
- Download the private key

### 3. Configure and Run

```bash
# Copy example configuration
cp example.env .env

# Edit .env with your settings:
# - GITHUB_TOKEN (for token auth) OR
# - GITHUB_APP_PRIVATE_KEY + GITHUB_APP_INSTALLATION_ID (for app auth)
# - TARGET_REPO=owner/repo
# - AI_BACKEND=opencode (free) or claude (requires API key)

# Start with OpenCode (free, default)
docker compose up -d

# OR start with Claude Code (requires CLAUDE_API_KEY)
docker compose -f docker-compose.claude.yml up -d

# View logs
docker compose logs -f agntpr
```

### 4. Test It

Open an issue in your target repo and mention the agent:

```
@your-org-bot Can you add input validation to the login form?
```

The agent will respond with a plan and wait for your approval.

## Key Features

### Intelligent Clarification

Agent detects unclear requirements and asks follow-up questions:

- Ambiguous issues: "Before I start, I have a question: ..."
- User questions: Answers technical questions about the codebase
- Waits for response before planning (stays in "new" state)

### Transparent Planning

Plans are posted as full text in GitHub comments, not hidden in files:

- Problem analysis with edge cases and assumptions
- Detailed implementation steps with specific file changes
- Test strategy with concrete examples
- Risk assessment and clarification needs

### Flexible AI Backends

- **OpenCode** - Free and open-source, uses Kimi K2.5 by default
- **Claude Code** - Anthropic's official CLI with Sonnet/Opus/Haiku

### GitHub App Support

Enterprise-ready authentication with automatic token refresh:

- Works with GitHub Apps (recommended for organizations)
- Fallback to personal access tokens (simpler for individuals)
- Tokens auto-refresh every hour

## 🚧 Experimental

AgntPR is young and opinionated:

- Expect rough edges
- Strong opinions about TDD methodology
- Built for developers who value structured process
- Fork-based workflow adds latency but maximizes safety

## Contributing

AgntPR is minimal by design.

If you love Go, GitHub workflows, and AI that knows its place — join
in. Fork, hack, PR. Keep it simple.

See [CLAUDE.md](CLAUDE.md) for architecture details.

## Architecture

Built with Go for reliability and simplicity:

- 🧠 **AI Backends** - [OpenCode](https://opencode.ai) (free) or
  [Claude Code](https://claude.ai/code)
- 🐙 **GitHub Integration** - GitHub CLI (`gh`) for API operations
- 🗄️ **State Management** - SQLite for persistence
- 🔄 **State Machine** - Explicit workflow states (new → planning →
  plan_review → implementing → pr_created → pr_review → done)
- 🍴 **Fork-Based Safety** - All work happens in forks, PRs from fork
  to upstream

### Deployment Options

Three Docker Compose configurations:

- **`docker-compose.yml`** - OpenCode backend (free, pre-built image)
- **`docker-compose.claude.yml`** - Claude Code backend (requires API
  key, pre-built image)
- **`docker-compose.local.yml`** - Local development (builds from
  source, debug mode)

### Custom OpenCode Models

OpenCode uses free Kimi K2.5 by default. To use other providers:

1. Authenticate locally: `opencode auth login`
2. Copy config: `~/.config/opencode/` → `./opencode-config/`
3. Copy auth: `~/.local/share/opencode/auth.json` →
   `./opencode-auth.json`
4. Uncomment volume mounts in `docker-compose.yml`
5. Set `OPENCODE_MODEL` env var (e.g., `anthropic/claude-sonnet-4-5`)

See `example.env` for all configuration options.

## License

MIT
