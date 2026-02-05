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

1. Mention `@ai-r-sentry` on a GitHub issue
2. Agent forks your repo, creates a plan, waits for approval
3. Agent implements with TDD, opens PR from fork
4. You review, request changes, merge

```bash
# That's it. No setup in your repo.
docker-compose up -d
```

Skip planning for trivial issues:
`@ai-r-sentry just implement this directly`

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

- 🧠 [Claude Code](https://claude.ai/code) - The brain
- 🐙 GitHub CLI - The hands
- 🗄️ SQLite - The memory

## License

MIT
