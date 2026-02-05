# 🤖 AgntPR

An AI agent that lives in your GitHub. Watches issues. Ships PRs.

## Why AgntPR?

AI coding assistants sit in your IDE waiting for commands. Copy-paste
workflows. Context switching. Babysitting.

AgntPR takes a different approach:

- **No IDE plugins.** Just GitHub issues.
- **No copy-paste prompts.** Natural language in comments.
- **No babysitting.** Agent runs autonomously.
- **No direct repo access.** Fork-based isolation.
- **Plan-first workflow.** See the strategy before code changes.
- **TDD enforcement.** RED-GREEN-REFACTOR cycle baked in.
- **You stay in control.** Approve plans. Review PRs. Merge when ready.

## How It Works

1. Mention the agent's GitHub account on an issue
2. Agent forks your repo, creates a plan, waits for approval
3. Agent implements with TDD, opens PR from fork
4. You review, request changes, merge

```bash
# 1. Create a dedicated GitHub account for the agent
# 2. Generate a fine-grained token with repo access (read/write)
# 3. Copy example.env to .env and configure

cp example.env .env
# Edit .env with your GITHUB_TOKEN and TARGET_REPO

# Option A: OpenCode (free, default - uses pre-built image)
docker compose up -d

# Option B: Claude Code (requires CLAUDE_API_KEY - uses pre-built image)
docker compose -f docker-compose.claude.yml up -d

# Option C: Local development (builds from source with debug mode)
docker compose -f docker-compose.local.yml up --build
```

**⚠️ Security**: Use a dedicated GitHub account with a fine-grained
personal access token. Never use your personal account token. Scope
the token to only the repositories the agent should access.

Skip planning for trivial issues by mentioning "just implement this
directly" in your comment.

## 🚧 Experimental

AgntPR is young and opinionated.

- Expect rough edges
- Expect strong opinions about TDD
- Built for developers who value process

## Contributing

AgntPR is minimal by design.

If you love Go, GitHub workflows, and AI that knows its place — join
in. Fork, hack, PR. Keep it simple.

See [CLAUDE.md](CLAUDE.md) for architecture details.

## Built With

- 🧠 [OpenCode](https://opencode.ai) or [Claude
  Code](https://claude.ai/code) - The brain
- 🐙 GitHub CLI - The hands
- 🗄️ SQLite - The memory

### AI Backend Options

AgntPR supports two backends:

- **OpenCode** (default) - Free and open-source alternative
- **Claude Code** - Anthropic's official CLI

#### Docker Compose Configurations

Three configurations are provided for different use cases:

- **`docker-compose.yml`** - Production with OpenCode (pre-built
  image)
- **`docker-compose.claude.yml`** - Production with Claude Code
  (pre-built image)
- **`docker-compose.local.yml`** - Local development (builds from
  source)

See `example.env` for configuration options.

#### Custom OpenCode Providers

By default, OpenCode uses the free Kimi K2.5 model. To use other
providers:

1. Run `opencode auth login` locally to configure your provider
2. Copy `~/.config/opencode/` to `./opencode-config/`
3. Copy `~/.local/share/opencode/auth.json` to `./opencode-auth.json`
4. Uncomment the volume mounts in `docker-compose.yml`
5. Set `OPENCODE_MODEL` to your desired model (e.g.,
   `anthropic/claude-sonnet-4-5`)

## License

MIT
