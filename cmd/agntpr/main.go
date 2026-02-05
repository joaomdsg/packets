package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joaomdsg/agntpr/internal/agent"
	"github.com/joaomdsg/agntpr/internal/config"
	"github.com/joaomdsg/agntpr/internal/db"
	"github.com/joaomdsg/agntpr/internal/fork"
	"github.com/joaomdsg/agntpr/internal/orchestrator"
	"github.com/joaomdsg/agntpr/internal/watcher"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	if err := run(ctx, cfg); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	ghCli := watcher.NewGHCli()

	agentUsername, err := ghCli.GetAuthenticatedUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authenticated user: %w", err)
	}

	log.Printf("agntpr starting as @%s, watching %s", agentUsername, cfg.TargetRepo)

	if cfg.ResetDB {
		log.Printf("RESET_DB set, removing database at %s", cfg.DatabasePath)
		if err := os.Remove(cfg.DatabasePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove database: %w", err)
		}
	}

	agentMention := "@" + agentUsername

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()

	w := watcher.New(
		ghCli, cfg.RepoOwner, cfg.RepoName, agentMention,
	)

	gitCli := fork.NewGitCli()
	ghAdapter := &ghForkAdapter{cli: ghCli}
	fm := fork.NewManager(
		gitCli, ghAdapter, cfg.WorkDir,
		"upstream", "origin", agentUsername,
	)

	claudeRunner := agent.NewClaudeRunner(30*time.Minute, cfg.ClaudeModel)
	agentWrapper := &agentAdapter{
		invoker: agent.NewInvoker(claudeRunner),
		debug:   cfg.Debug,
	}

	log.Printf("using Claude model: %s", cfg.ClaudeModel)
	if cfg.Debug {
		log.Println("DEBUG mode enabled")
	}

	orch := orchestrator.New(
		database, w, fm, agentWrapper,
		cfg.RepoOwner, cfg.RepoName,
	)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	log.Printf("polling every %s", cfg.PollInterval)

	// Initial poll
	if err := orch.ProcessIssues(ctx); err != nil {
		log.Printf("poll error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("shutdown complete")
			return nil
		case <-ticker.C:
			if err := orch.ProcessIssues(ctx); err != nil {
				log.Printf("poll error: %v", err)
			}
		}
	}
}

type agentAdapter struct {
	invoker *agent.Invoker
	debug   bool
}

func (a *agentAdapter) Plan(
	ctx context.Context, workDir string, issue *db.Issue,
) (string, error) {
	// Determine if this is a revision by checking plan history
	// This will be set when called from orchestrator with plan version > 1
	isRevision := false
	planVersion := 1

	// Note: We'd need to pass this info from orchestrator, but for now
	// the context file will contain the history

	req := &agent.Request{
		WorkDir:     workDir,
		IssueNumber: issue.Number,
		IssueTitle:  issue.Title,
		IssueBody:   issue.Body,
		BaseBranch:  "main", // TODO: Make this configurable
		IsRevision:  isRevision,
		PlanVersion: planVersion,
	}

	if a.debug {
		prompt := agent.BuildPlanningPrompt(req)
		log.Printf("[DEBUG] Plan() prompt:\n%s", prompt)
	}

	result, err := a.invoker.Plan(ctx, req)
	if err != nil {
		return "", err
	}
	if !result.Success {
		return "", &agentError{msg: result.Error}
	}

	if a.debug {
		log.Printf("[DEBUG] Plan() result: success=%v, output_len=%d", result.Success, len(result.Output))
	}

	return result.Output, nil
}

func (a *agentAdapter) Implement(
	ctx context.Context, workDir string, issue *db.Issue, plan string,
) error {
	req := &agent.Request{
		WorkDir:     workDir,
		IssueNumber: issue.Number,
		IssueTitle:  issue.Title,
		IssueBody:   issue.Body,
		Plan:        plan,
		BaseBranch:  "main", // TODO: Make this configurable
	}

	if a.debug {
		prompt := agent.BuildImplementationPrompt(req)
		log.Printf("[DEBUG] Implement() prompt:\n%s", prompt)
	}

	result, err := a.invoker.Implement(ctx, req)
	if err != nil {
		return err
	}
	if !result.Success {
		return &agentError{msg: result.Error}
	}

	if a.debug {
		log.Printf("[DEBUG] Implement() result: success=%v", result.Success)
	}

	return nil
}

func (a *agentAdapter) RespondToReview(
	ctx context.Context, workDir, comment string,
) error {
	req := &agent.Request{
		WorkDir:       workDir,
		ReviewComment: comment,
		BaseBranch:    "main", // TODO: Make this configurable
	}

	if a.debug {
		prompt := agent.BuildReviewResponsePrompt(req)
		log.Printf("[DEBUG] RespondToReview() prompt:\n%s", prompt)
	}

	result, err := a.invoker.RespondToReview(ctx, req)
	if err != nil {
		return err
	}
	if !result.Success {
		return &agentError{msg: result.Error}
	}

	if a.debug {
		log.Printf("[DEBUG] RespondToReview() result: success=%v", result.Success)
	}

	return nil
}

func (a *agentAdapter) SummarizeChanges(
	ctx context.Context, workDir string,
) (string, error) {
	if a.debug {
		prompt := agent.BuildSummaryPrompt()
		log.Printf("[DEBUG] SummarizeChanges() prompt:\n%s", prompt)
	}

	result, err := a.invoker.SummarizeChanges(ctx, workDir)
	if err != nil {
		return "", err
	}
	if !result.Success {
		return "", &agentError{msg: result.Error}
	}

	if a.debug {
		log.Printf("[DEBUG] SummarizeChanges() result: success=%v, output_len=%d", result.Success, len(result.Output))
	}

	return result.Output, nil
}

func (a *agentAdapter) AnswerQuestion(
	ctx context.Context, workDir, question, issueContext string,
) (string, error) {
	req := &agent.AnswerRequest{
		WorkDir:      workDir,
		Question:     question,
		IssueContext: issueContext,
	}

	if a.debug {
		prompt := agent.BuildAnswerPrompt(req)
		log.Printf("[DEBUG] AnswerQuestion() prompt:\n%s", prompt)
	}

	result, err := a.invoker.AnswerQuestion(ctx, req)
	if err != nil {
		return "", err
	}
	if !result.Success {
		return "", &agentError{msg: result.Error}
	}

	if a.debug {
		log.Printf("[DEBUG] AnswerQuestion() result: success=%v, output_len=%d", result.Success, len(result.Output))
	}

	return result.Output, nil
}

func (a *agentAdapter) EvaluateIntent(
	ctx context.Context, issueTitle, issueBody string, labels []string, comments []string,
) (*orchestrator.Intent, error) {
	req := &agent.IntentRequest{
		IssueTitle: issueTitle,
		IssueBody:  issueBody,
		Labels:     labels,
		Comments:   comments,
	}

	if a.debug {
		prompt := agent.BuildIntentPrompt(req)
		log.Printf("[DEBUG] EvaluateIntent() prompt:\n%s", prompt)
	}

	result, err := a.invoker.EvaluateIntent(ctx, req)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, &agentError{msg: result.Error}
	}

	if a.debug {
		log.Printf("[DEBUG] EvaluateIntent() raw output:\n%s", result.Output)
	}

	// Parse JSON from result
	intent := &orchestrator.Intent{}
	if err := parseIntentJSON(result.Output, intent); err != nil {
		return nil, fmt.Errorf("parse intent failed: %w", err)
	}

	if a.debug {
		log.Printf("[DEBUG] EvaluateIntent() parsed: skip_planning=%v, skip_approval=%v, is_approval=%v, is_revision=%v, is_question=%v, needs_clarify=%v",
			intent.SkipPlanning, intent.SkipApproval, intent.IsApproval, intent.IsRevision, intent.IsQuestion, intent.NeedsClarify)
	}

	return intent, nil
}

func parseIntentJSON(output string, intent *orchestrator.Intent) error {
	// Extract JSON from output (may be wrapped in markdown code block)
	output = strings.TrimSpace(output)
	if strings.HasPrefix(output, "```") {
		lines := strings.Split(output, "\n")
		var jsonLines []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				jsonLines = append(jsonLines, line)
			}
		}
		output = strings.Join(jsonLines, "\n")
	}

	var parsed agent.IntentResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		// Truncate output for error message
		preview := output
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return fmt.Errorf("%w (got: %q)", err, preview)
	}

	intent.SkipPlanning = parsed.SkipPlanning
	intent.SkipApproval = parsed.SkipApproval
	intent.IsApproval = parsed.IsApproval
	intent.IsRevision = parsed.IsRevision
	intent.IsQuestion = parsed.IsQuestion
	intent.Feedback = parsed.Feedback
	intent.NeedsClarify = parsed.NeedsClarify
	intent.Question = parsed.Question

	return nil
}

type agentError struct {
	msg string
}

func (e *agentError) Error() string {
	return e.msg
}

type ghForkAdapter struct {
	cli *watcher.GHCli
}

func (g *ghForkAdapter) ForkRepo(ctx context.Context, owner, repo string) error {
	return g.cli.ForkRepo(ctx, owner, repo)
}

func (g *ghForkAdapter) CloneRepo(
	ctx context.Context, owner, repo, dest string,
) error {
	return g.cli.CloneRepo(ctx, owner, repo, dest)
}
