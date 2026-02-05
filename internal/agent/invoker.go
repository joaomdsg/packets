package agent

import (
	"context"
	"fmt"
	"strings"
)

type Runner interface {
	Run(ctx context.Context, workDir, prompt string) (*Result, error)
}

type Request struct {
	WorkDir       string
	IssueNumber   int
	IssueTitle    string
	IssueBody     string
	Plan          string
	ReviewComment string
	BaseBranch    string
	IsRevision    bool
	PlanVersion   int
}

type Result struct {
	Output  string
	Success bool
	Error   string
}

type Invoker struct {
	runner Runner
}

func NewInvoker(runner Runner) *Invoker {
	return &Invoker{runner: runner}
}

func (i *Invoker) Plan(ctx context.Context, req *Request) (*Result, error) {
	prompt := BuildPlanningPrompt(req)
	return i.runner.Run(ctx, req.WorkDir, prompt)
}

func (i *Invoker) Implement(ctx context.Context, req *Request) (*Result, error) {
	prompt := BuildImplementationPrompt(req)
	return i.runner.Run(ctx, req.WorkDir, prompt)
}

func (i *Invoker) RespondToReview(
	ctx context.Context, req *Request,
) (*Result, error) {
	prompt := BuildReviewResponsePrompt(req)
	return i.runner.Run(ctx, req.WorkDir, prompt)
}

type IntentRequest struct {
	IssueTitle string
	IssueBody  string
	Labels     []string
	Comments   []string
}

type AnswerRequest struct {
	WorkDir      string
	Question     string
	IssueContext string
}

type IntentResult struct {
	SkipPlanning bool   `json:"skip_planning"`
	SkipApproval bool   `json:"skip_approval"`
	IsApproval   bool   `json:"is_approval"`
	IsRevision   bool   `json:"is_revision"`
	IsQuestion   bool   `json:"is_question"`
	Feedback     string `json:"feedback"`
	NeedsClarify bool   `json:"needs_clarification"`
	Question     string `json:"question"`
	Confidence   string `json:"confidence"` // "high", "medium", "low"
}

func (i *Invoker) EvaluateIntent(
	ctx context.Context, req *IntentRequest,
) (*Result, error) {
	prompt := BuildIntentPrompt(req)
	return i.runner.Run(ctx, "", prompt)
}

func (i *Invoker) SummarizeChanges(
	ctx context.Context, workDir string,
) (*Result, error) {
	prompt := BuildSummaryPrompt()
	return i.runner.Run(ctx, workDir, prompt)
}

func (i *Invoker) AnswerQuestion(
	ctx context.Context, req *AnswerRequest,
) (*Result, error) {
	prompt := BuildAnswerPrompt(req)
	return i.runner.Run(ctx, req.WorkDir, prompt)
}

func BuildPlanningPrompt(req *Request) string {
	var b strings.Builder

	b.WriteString("Create an implementation plan for a GitHub issue.\n\n")

	b.WriteString("**Context**: Read `.agntpr-context.md` for full issue details, comments, and plan history. ")
	b.WriteString("Do not commit this file.\n\n")

	if req.IsRevision {
		b.WriteString(fmt.Sprintf("**Plan v%d** - Previous plan(s) were rejected. ", req.PlanVersion))
		b.WriteString("Address all feedback from the context file.\n\n")
	}

	b.WriteString("## Plan Structure\n\n")

	b.WriteString("### 1. Problem Analysis\n")
	b.WriteString("- What needs to change and why (2-3 sentences)\n")
	b.WriteString("- Edge cases to handle\n")
	b.WriteString("- Assumptions\n\n")

	b.WriteString("### 2. Implementation\n")
	b.WriteString("- Files to modify with specific functions (e.g., `internal/state/machine.go:Transition()`)\n")
	b.WriteString("- New files only if necessary\n")
	b.WriteString("- Key logic changes in plain language\n\n")

	b.WriteString("### 3. Test Strategy\n")
	b.WriteString("- Specific test scenarios (concrete examples)\n")
	b.WriteString("- Test types: unit tests for X, integration for Y\n")
	b.WriteString("- Existing tests to update\n\n")

	b.WriteString("### 4. Risks\n")
	b.WriteString("- Backward compatibility concerns\n")
	b.WriteString("- Performance implications\n")
	b.WriteString("- Areas needing maintainer clarification\n\n")

	b.WriteString("## Guidelines\n")
	b.WriteString("- Address ONLY what the issue requests\n")
	b.WriteString("- 300-500 words, specific not vague\n")
	b.WriteString("- Present tense (\"Add function\" not \"Will add\")\n")
	b.WriteString("- State your interpretation if unclear\n\n")

	b.WriteString("Success: A developer unfamiliar with this issue can implement from your plan.\n")

	return b.String()
}

func BuildImplementationPrompt(req *Request) string {
	var b strings.Builder

	b.WriteString("Implement the GitHub issue using Test-Driven Development.\n\n")

	b.WriteString("**Context**: Read `.agntpr-context.md` for approved plan. Do not commit this file.\n\n")

	b.WriteString("## TDD Cycle\n\n")

	b.WriteString("For each feature:\n\n")

	b.WriteString("**1. RED** - Write ONE failing test for smallest next piece\n")
	b.WriteString("- Run and verify it fails for the right reason\n\n")

	b.WriteString("**2. GREEN** - Write simplest code to pass\n")
	b.WriteString("- Don't add extra features\n")
	b.WriteString("- Run ALL tests (no regressions)\n\n")

	b.WriteString("**3. REFACTOR** - Improve clarity\n")
	b.WriteString("- Follow existing project patterns\n")
	b.WriteString("- Keep tests passing\n\n")

	b.WriteString("**4. REPEAT** until plan complete\n\n")

	b.WriteString("## Completion Checklist\n\n")

	b.WriteString("Before finishing:\n")
	b.WriteString("```bash\n")
	b.WriteString("go test ./...        # All tests pass\n")
	b.WriteString("go test -cover ./... # Check coverage\n")
	b.WriteString("git diff             # No debug code\n")
	b.WriteString("```\n\n")

	b.WriteString("Verify:\n")
	b.WriteString("- All plan requirements implemented\n")
	b.WriteString("- All tests pass\n")
	b.WriteString("- No debug/commented code\n")
	b.WriteString("- Code follows project patterns\n\n")

	b.WriteString("## Scope\n\n")

	b.WriteString("You're working on a fork. Stay within approved plan.\n")
	b.WriteString("Tools: `brew install <package>`, `docker` command\n")
	b.WriteString("Timeout: 30 minutes\n\n")

	b.WriteString("## Commits\n\n")

	b.WriteString("Commit when logical units of work are complete:\n")
	b.WriteString("- After completing a feature or significant refactor\n")
	b.WriteString("- When tests pass and code is clean\n")
	b.WriteString("- Format: `<verb> <component>: <what changed>`\n")
	b.WriteString("- Example: `Add state: implement transition validation`\n\n")

	b.WriteString("If unclear: document your interpretation in commit message.\n")

	return b.String()
}

func BuildReviewResponsePrompt(req *Request) string {
	var b strings.Builder

	b.WriteString("Address PR review feedback from maintainer.\n\n")

	b.WriteString("**Context**: Read `.agntpr-context.md` for full issue, implementation, and PR comments. ")
	b.WriteString("Do not commit this file.\n\n")

	b.WriteString("## Review Feedback\n\n")
	b.WriteString(req.ReviewComment)
	b.WriteString("\n\n")

	b.WriteString("## Implementation\n\n")

	b.WriteString("Use TDD cycle:\n\n")

	b.WriteString("**1. RED** - Update/add tests for new requirements\n")
	b.WriteString("**2. GREEN** - Implement requested changes\n")
	b.WriteString("**3. REFACTOR** - Clean up if needed\n\n")

	b.WriteString("## Scope Concerns\n\n")

	b.WriteString("If request seems out of scope or conflicts with plan:\n")
	b.WriteString("- Implement it anyway (maintainer has authority)\n")
	b.WriteString("- Document concern in commit message body\n\n")

	b.WriteString("Example:\n")
	b.WriteString("```\n")
	b.WriteString("Review: add feature X\n\n")
	b.WriteString("Note: This extends beyond original issue scope.\n")
	b.WriteString("```\n\n")

	b.WriteString("## Verification\n\n")

	b.WriteString("Before finishing:\n")
	b.WriteString("- All tests pass\n")
	b.WriteString("- Feedback fully addressed\n")
	b.WriteString("- No debug code remains\n\n")

	b.WriteString("## Commits\n\n")

	b.WriteString("Commit when changes are complete:\n")
	b.WriteString("- Format: `Review: <what changed>`\n")
	b.WriteString("- Example: `Review: add error handling for nil pointer`\n")
	b.WriteString("- If out of scope, document concern in commit body\n\n")

	b.WriteString("Tools: `brew install <package>`, `docker` command\n")

	return b.String()
}

func BuildAnswerPrompt(req *AnswerRequest) string {
	var b strings.Builder

	b.WriteString("Answer a maintainer's question. DO NOT make code changes.\n\n")

	b.WriteString("**Context**: Read `.agntpr-context.md` for background. Do not commit this file.\n\n")

	b.WriteString("## Question\n\n")
	b.WriteString(req.Question)
	b.WriteString("\n\n")

	b.WriteString("## Process\n\n")

	b.WriteString("1. Search codebase (grep/find)\n")
	b.WriteString("2. Read relevant files\n")
	b.WriteString("3. Check context file\n")
	b.WriteString("4. Formulate specific answer\n\n")

	b.WriteString("## Answer Format\n\n")

	b.WriteString("Direct answer (1-2 sentences) + supporting details + code references\n\n")

	b.WriteString("Example:\n")
	b.WriteString("```\n")
	b.WriteString("The state machine transitions from `planning` to `plan_review` when ")
	b.WriteString("the agent completes creating a plan.\n\n")

	b.WriteString("This happens in internal/orchestrator/orchestrator.go:308 where ")
	b.WriteString("`transitionTo(ctx, issue, state.EventPlanComplete)` is called.\n")
	b.WriteString("```\n\n")

	b.WriteString("## Guidelines\n\n")

	b.WriteString("- Length: 2-4 sentences (50-150 words)\n")
	b.WriteString("- Reference: file.go:line or function names\n")
	b.WriteString("- Uncertainty: Say so if unsure\n")
	b.WriteString("- Scope: Answer only what was asked\n")
	b.WriteString("- Start directly with answer (no preamble)\n")

	return b.String()
}

func BuildSummaryPrompt() string {
	var b strings.Builder

	b.WriteString("Create a Pull Request summary for maintainers.\n\n")

	b.WriteString("## Process\n\n")

	b.WriteString("```bash\n")
	b.WriteString("git log origin/main..HEAD --oneline  # Review commits\n")
	b.WriteString("git diff origin/main --stat           # See changed files\n")
	b.WriteString("git diff origin/main                  # Review changes\n")
	b.WriteString("```\n\n")

	b.WriteString("## Format\n\n")

	b.WriteString("Write 3-7 bullet points covering:\n")
	b.WriteString("- Functionality added/modified/fixed\n")
	b.WriteString("- Components affected (mention key files)\n")
	b.WriteString("- Test coverage added\n\n")

	b.WriteString("Example:\n")
	b.WriteString("```\n")
	b.WriteString("- Added transition validation to state machine (`internal/state/machine.go`)\n")
	b.WriteString("- Updated orchestrator to handle invalid transitions gracefully\n")
	b.WriteString("- Added unit tests for all transition edge cases (12 new tests)\n")
	b.WriteString("- Fixed bug where errored issues could re-enter planning state\n")
	b.WriteString("```\n\n")

	b.WriteString("## Style\n\n")

	b.WriteString("- **Tense**: Past tense (\"Added\", \"Fixed\", \"Updated\")\n")
	b.WriteString("- **Length**: 10-20 words per bullet\n")
	b.WriteString("- **Focus**: WHAT and WHY, not HOW\n")
	b.WriteString("- **Specificity**: Mention key files but stay readable\n\n")

	b.WriteString("## Output\n\n")

	b.WriteString("ONLY the bullet list. No headers, no explanatory text.\n")
	b.WriteString("Start each line with `- ` (dash and space).\n")

	return b.String()
}

func BuildIntentPrompt(req *IntentRequest) string {
	var b strings.Builder

	b.WriteString("Analyze maintainer intent from GitHub comments.\n\n")

	b.WriteString("**Issue:** ")
	b.WriteString(req.IssueTitle)
	if len(req.Labels) > 0 {
		b.WriteString(" [")
		b.WriteString(strings.Join(req.Labels, ", "))
		b.WriteString("]")
	}
	b.WriteString("\n\n")

	b.WriteString(req.IssueBody)
	b.WriteString("\n\n")

	if len(req.Comments) > 0 {
		b.WriteString("**Comments:**\n")
		for i, c := range req.Comments {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Task\n\n")
	b.WriteString("Classify the most recent comment into ONE intent category:\n\n")

	b.WriteString("**skip_planning**: Skip plan, implement directly (\"just implement\", \"straightforward\")\n")
	b.WriteString("- Also set skip_approval = true\n\n")

	b.WriteString("**is_approval**: Approve proposed plan (\"approved\", \"LGTM\", \"looks good\")\n\n")

	b.WriteString("**is_revision**: Request changes (\"please change X\", \"instead do Y\")\n")
	b.WriteString("- Set feedback field with requested changes\n\n")

	b.WriteString("**is_question**: Asking for information (ends with '?', \"how does\", \"why did\")\n")
	b.WriteString("- Set question field with extracted question\n\n")

	b.WriteString("**needs_clarification**: Intent unclear or <70% confidence\n")
	b.WriteString("- Set question field asking what you need to know\n\n")

	b.WriteString("## Priority Rules\n\n")
	b.WriteString("If multiple intents: question > revision > approval > skip_planning\n")
	b.WriteString("Casual comments (\"thanks\", \"ok\"): all bools false\n\n")

	b.WriteString("## Output\n\n")
	b.WriteString("Single-line JSON only. No markdown, no explanation.\n\n")

	b.WriteString("Format:\n")
	b.WriteString(`{"skip_planning": false, "skip_approval": false, "is_approval": false, `)
	b.WriteString(`"is_revision": false, "is_question": false, "feedback": "", `)
	b.WriteString(`"needs_clarification": false, "question": "", "confidence": "high"}`)
	b.WriteString("\n\n")

	b.WriteString("confidence: \"high\" (>90%), \"medium\" (70-90%), \"low\" (<70%)\n\n")

	b.WriteString("Examples:\n")
	b.WriteString("\"Looks good!\" → ")
	b.WriteString(`{"skip_planning": false, "skip_approval": false, "is_approval": true, `)
	b.WriteString(`"is_revision": false, "is_question": false, "feedback": "", `)
	b.WriteString(`"needs_clarification": false, "question": "", "confidence": "high"}`)
	b.WriteString("\n\n")

	b.WriteString("\"Please add error handling\" → ")
	b.WriteString(`{"skip_planning": false, "skip_approval": false, "is_approval": false, `)
	b.WriteString(`"is_revision": true, "is_question": false, "feedback": "Add error handling", `)
	b.WriteString(`"needs_clarification": false, "question": "", "confidence": "high"}`)
	b.WriteString("\n\n")

	b.WriteString("Requirements:\n")
	b.WriteString("- Valid JSON on single line\n")
	b.WriteString("- Only ONE intent true (except skip_planning + skip_approval)\n")
	b.WriteString("- If needs_clarification/is_revision/is_question = true, set corresponding field\n")
	b.WriteString("- Escape string values properly\n\n")

	b.WriteString("Begin with { and end with }. Nothing else.\n")

	return b.String()
}
