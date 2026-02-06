package orchestrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	issuecontext "github.com/joaomdsg/agntpr/internal/context"
	"github.com/joaomdsg/agntpr/internal/db"
	"github.com/joaomdsg/agntpr/internal/state"
	"github.com/joaomdsg/agntpr/internal/watcher"
)

type Watcher interface {
	FetchMentionedIssues(ctx context.Context) ([]*watcher.Issue, error)
	FetchIssueComments(ctx context.Context, issueNum int) ([]*watcher.Comment, error)
	FetchOpenPRs(ctx context.Context) ([]*watcher.PR, error)
	FetchPRComments(ctx context.Context, prNum int) ([]*watcher.Comment, error)
	FetchAllPRComments(ctx context.Context, prNum int) ([]*watcher.Comment, error)
	FilterMentionedComments(comments []*watcher.Comment) []*watcher.Comment
	PostComment(ctx context.Context, num int, body string) (int64, error)
	CreatePR(ctx context.Context, title, body, head, base string) (*watcher.PR, error)
	GetPR(ctx context.Context, prNum int) (*watcher.PR, error)
	Mention() string
}

type ForkManager interface {
	SetupWorkDir(
		ctx context.Context, owner, repo string, issueNum int,
	) (string, error)
	CreateBranch(ctx context.Context, workDir string, issueNum int) (string, error)
	SyncWithUpstream(ctx context.Context, workDir, baseBranch string) error
	PushBranch(ctx context.Context, workDir, branch string, force bool) error
	HasChanges(ctx context.Context, workDir, baseBranch string) (bool, error)
}

type Intent struct {
	SkipPlanning bool
	SkipApproval bool
	IsApproval   bool
	IsRevision   bool
	IsQuestion   bool
	Feedback     string
	NeedsClarify bool
	Question     string
}

type Agent interface {
	Plan(ctx context.Context, workDir string, issue *db.Issue) (string, error)
	Implement(
		ctx context.Context, workDir string, issue *db.Issue, plan string,
	) error
	SummarizeChanges(ctx context.Context, workDir string) (string, error)
	AnswerQuestion(
		ctx context.Context, workDir, question, issueContext string,
	) (string, error)
	RespondToReview(ctx context.Context, workDir, comment string) error
	EvaluateIntent(ctx context.Context, issueTitle, issueBody string, labels []string, comments []string) (*Intent, error)
}

type Orchestrator struct {
	db      *db.DB
	watcher Watcher
	fork    ForkManager
	agent   Agent
	owner   string
	repo    string
}

func New(
	database *db.DB,
	watcher Watcher,
	fork ForkManager,
	agent Agent,
	owner, repo string,
) *Orchestrator {
	return &Orchestrator{
		db:      database,
		watcher: watcher,
		fork:    fork,
		agent:   agent,
		owner:   owner,
		repo:    repo,
	}
}

func (o *Orchestrator) ProcessIssues(ctx context.Context) error {
	issues, err := o.watcher.FetchMentionedIssues(ctx)
	if err != nil {
		return fmt.Errorf("fetch issues failed: %w", err)
	}

	for _, ghIssue := range issues {
		if err := o.processIssue(ctx, ghIssue); err != nil {
			log.Printf("error processing issue #%d: %v", ghIssue.Number, err)
			o.handleProcessingError(ctx, ghIssue, err)
		}
	}

	return nil
}

func (o *Orchestrator) handleProcessingError(
	ctx context.Context, ghIssue *watcher.Issue, processErr error,
) {
	// Post error comment
	errMsg := fmt.Sprintf(
		"❌ **Error encountered while processing this issue.**\n\n"+
			"```\n%s\n```\n\n"+
			"I've paused work on this issue. "+
			"Please fix the issue and re-assign me, or close this issue.",
		processErr.Error())

	if _, err := o.watcher.PostComment(ctx, ghIssue.Number, errMsg); err != nil {
		log.Printf("warning: failed to post error comment: %v", err)
	}

	// Mark issue as errored in database
	existing, err := o.db.GetIssueByGitHubID(ctx, ghIssue.ID)
	if err != nil {
		log.Printf("warning: failed to get issue for error state: %v", err)
		return
	}

	if err := o.db.UpdateIssueState(ctx, existing.ID, db.StateErrored); err != nil {
		log.Printf("warning: failed to mark issue as errored: %v", err)
	}
}

func (o *Orchestrator) processIssue(
	ctx context.Context, ghIssue *watcher.Issue,
) error {
	existing, err := o.db.GetIssueByGitHubID(ctx, ghIssue.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("db lookup failed: %w", err)
	}

	if existing != nil {
		return o.continueIssue(ctx, existing)
	}

	return o.startNewIssue(ctx, ghIssue)
}

func (o *Orchestrator) startNewIssue(
	ctx context.Context, ghIssue *watcher.Issue,
) error {
	log.Printf("starting new issue #%d: %s", ghIssue.Number, ghIssue.Title)

	issue := &db.Issue{
		GitHubID: ghIssue.ID,
		Number:   ghIssue.Number,
		Title:    ghIssue.Title,
		Body:     ghIssue.Body,
		State:    db.StateNew,
	}

	// Evaluate intent from issue body and existing comments
	existingComments, err := o.getCommentBodies(ctx, ghIssue.Number)
	if err != nil {
		log.Printf("warning: failed to get existing comments: %v", err)
	}

	var labels []string
	for _, l := range ghIssue.Labels {
		labels = append(labels, l.Name)
	}

	intent, err := o.agent.EvaluateIntent(ctx, ghIssue.Title, ghIssue.Body, labels, existingComments)
	if err != nil {
		log.Printf("warning: failed to evaluate intent: %v", err)
	} else {
		issue.SkipPlanning = intent.SkipPlanning
		issue.SkipApproval = intent.SkipApproval

		// Handle questions - answer and wait for follow-up
		if intent.IsQuestion {
			log.Printf("issue #%d: question detected", ghIssue.Number)
			issue.State = db.StateNew
			if err := o.db.CreateIssue(ctx, issue); err != nil {
				return fmt.Errorf("create issue failed: %w", err)
			}
			if err := o.answerQuestion(ctx, issue, intent.Question); err != nil {
				log.Printf("warning: failed to answer question: %v", err)
			}
			return nil // Stay in "new" - wait for follow-up
		}

		// Handle clarifications - ask and wait for response
		if intent.NeedsClarify {
			log.Printf("issue #%d: clarification needed", ghIssue.Number)
			issue.State = db.StateNew
			if err := o.db.CreateIssue(ctx, issue); err != nil {
				return fmt.Errorf("create issue failed: %w", err)
			}
			if _, err := o.watcher.PostComment(ctx, ghIssue.Number,
				fmt.Sprintf("👋 Before I start, I have a question:\n\n%s", intent.Question)); err != nil {
				log.Printf("warning: failed to post clarifying question: %v", err)
			}
			return nil // Stay in "new" - wait for response
		}
	}

	if err := o.db.CreateIssue(ctx, issue); err != nil {
		return fmt.Errorf("create issue failed: %w", err)
	}

	// Skip planning if requested
	if issue.SkipPlanning {
		log.Printf("issue #%d: skipping planning phase per maintainer request", issue.Number)
		return o.transitionTo(ctx, issue, state.EventSkipPlanning)
	}

	return o.transitionTo(ctx, issue, state.EventStartPlanning)
}

func (o *Orchestrator) continueIssue(ctx context.Context, issue *db.Issue) error {
	machine := state.NewMachine(issue.State)

	if machine.IsTerminal() {
		return nil
	}

	switch issue.State {
	case db.StateNew:
		// Check for new comments that might be responses to our questions
		comments, err := o.watcher.FetchIssueComments(ctx, issue.Number)
		if err != nil {
			return fmt.Errorf("fetch comments failed: %w", err)
		}

		// Filter for mentions (responses to our questions)
		mentioned := o.watcher.FilterMentionedComments(comments)
		if len(mentioned) == 0 {
			// No new mentions - stay in "new" state, wait for response
			return nil
		}

		// Get latest mentioned comment
		latest := mentioned[len(mentioned)-1]

		// Evaluate intent of the response
		intent, err := o.agent.EvaluateIntent(ctx, issue.Title, issue.Body, nil, []string{latest.Body})
		if err != nil {
			log.Printf("warning: failed to evaluate intent: %v", err)
			// Proceed to planning on evaluation failure
			return o.transitionTo(ctx, issue, state.EventStartPlanning)
		}

		// If still asking questions, answer and wait
		if intent.IsQuestion {
			return o.answerQuestion(ctx, issue, intent.Question)
		}

		// If needs more clarification, ask and wait
		if intent.NeedsClarify {
			if _, err := o.watcher.PostComment(ctx, issue.Number,
				fmt.Sprintf("👋 I have another question:\n\n%s", intent.Question)); err != nil {
				log.Printf("warning: failed to post question: %v", err)
			}
			return nil
		}

		// Response received - proceed to planning
		return o.transitionTo(ctx, issue, state.EventStartPlanning)
	case db.StatePlanning:
		return o.executePlanning(ctx, issue)
	case db.StatePlanReview:
		return o.checkForApproval(ctx, issue)
	case db.StateImplementing:
		return o.executeImplementation(ctx, issue)
	case db.StatePRCreated, db.StatePRReview:
		return o.processPRComments(ctx, issue)
	}

	return nil
}

func (o *Orchestrator) transitionTo(
	ctx context.Context, issue *db.Issue, event state.Event,
) error {
	oldState := issue.State
	machine := state.NewMachine(issue.State)

	newState, err := machine.Transition(event)
	if err != nil {
		return fmt.Errorf("transition failed: %w", err)
	}

	if err := o.db.UpdateIssueState(ctx, issue.ID, newState); err != nil {
		return fmt.Errorf("update state failed: %w", err)
	}
	issue.State = newState

	log.Printf("issue #%d: %s -> %s", issue.Number, oldState, newState)

	return o.executeState(ctx, issue)
}

func (o *Orchestrator) executeState(ctx context.Context, issue *db.Issue) error {
	switch issue.State {
	case db.StatePlanning:
		return o.executePlanning(ctx, issue)
	case db.StateImplementing:
		return o.executeImplementation(ctx, issue)
	}
	return nil
}

func (o *Orchestrator) executePlanning(ctx context.Context, issue *db.Issue) error {
	// Post status comment
	if _, err := o.watcher.PostComment(ctx, issue.Number,
		"🔬 **Started analyzing issue and creating implementation plan.**"); err != nil {
		log.Printf("warning: failed to post status comment: %v", err)
	}

	workDir, err := o.fork.SetupWorkDir(ctx, o.owner, o.repo, issue.Number)
	if err != nil {
		return fmt.Errorf("setup work dir failed: %w", err)
	}

	// Sync with upstream to ensure we're working with latest code
	if err := o.fork.SyncWithUpstream(ctx, workDir, "main"); err != nil {
		return fmt.Errorf("sync with upstream failed: %w", err)
	}

	if err := o.db.UpdateIssueBranch(ctx, issue.ID, "", workDir); err != nil {
		return fmt.Errorf("update work dir failed: %w", err)
	}

	if err := o.updateContextFile(ctx, issue, workDir); err != nil {
		log.Printf("warning: failed to update context file: %v", err)
	}

	plan, err := o.agent.Plan(ctx, workDir, issue)
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}

	// Get current plan version
	existingPlan, _ := o.db.GetLatestPlan(ctx, issue.ID)
	version := 1
	if existingPlan != nil {
		version = existingPlan.Version + 1
	}

	dbPlan := &db.Plan{
		IssueID: issue.ID,
		Version: version,
		Content: plan,
	}
	if err := o.db.CreatePlan(ctx, dbPlan); err != nil {
		return fmt.Errorf("save plan failed: %w", err)
	}

	// Post plan as comment
	planComment := fmt.Sprintf(
		"## Proposed Implementation Plan (v%d)\n\n%s\n\n"+
			"---\n"+
			"Reply with `%s approve` to approve this plan "+
			"and start implementation.\n"+
			"Reply with `%s revise: <feedback>` to request changes.",
		version, plan, o.watcher.Mention(), o.watcher.Mention())

	if _, err := o.watcher.PostComment(ctx, issue.Number, planComment); err != nil {
		return fmt.Errorf("post plan comment failed: %w", err)
	}

	return o.transitionTo(ctx, issue, state.EventPlanComplete)
}

func (o *Orchestrator) executeImplementation(
	ctx context.Context, issue *db.Issue,
) error {
	// Post implementation status
	statusMsg := "⚡ **Started implementation.**"
	if !issue.SkipPlanning {
		statusMsg = "⚡ **Plan approved. Started implementation.**"
	}
	if _, err := o.watcher.PostComment(ctx, issue.Number, statusMsg); err != nil {
		log.Printf("warning: failed to post status comment: %v", err)
	}

	// Get plan if planning wasn't skipped
	var planContent string
	if !issue.SkipPlanning {
		plan, err := o.db.GetLatestPlan(ctx, issue.ID)
		if err != nil {
			return fmt.Errorf("get plan failed: %w", err)
		}
		planContent = plan.Content
	}

	workDir := issue.WorkDir
	var err error
	if workDir == "" {
		workDir, err = o.fork.SetupWorkDir(ctx, o.owner, o.repo, issue.Number)
		if err != nil {
			return fmt.Errorf("setup work dir failed: %w", err)
		}
	}

	// Sync with upstream to ensure we're working with latest code
	if err := o.fork.SyncWithUpstream(ctx, workDir, "main"); err != nil {
		return fmt.Errorf("sync with upstream failed: %w", err)
	}

	branch, err := o.fork.CreateBranch(ctx, workDir, issue.Number)
	if err != nil {
		return fmt.Errorf("create branch failed: %w", err)
	}

	if err := o.db.UpdateIssueBranch(ctx, issue.ID, branch, workDir); err != nil {
		return fmt.Errorf("update branch failed: %w", err)
	}

	if err := o.updateContextFile(ctx, issue, workDir); err != nil {
		log.Printf("warning: failed to update context file: %v", err)
	}

	if err := o.agent.Implement(ctx, workDir, issue, planContent); err != nil {
		return fmt.Errorf("implementation failed: %w", err)
	}

	// Check if there are actual changes to push
	hasChanges, err := o.fork.HasChanges(ctx, workDir, "main")
	if err != nil {
		return fmt.Errorf("check changes failed: %w", err)
	}
	if !hasChanges {
		return fmt.Errorf("implementation produced no changes")
	}

	// Generate summary of changes
	summary, err := o.agent.SummarizeChanges(ctx, workDir)
	if err != nil {
		log.Printf("warning: failed to generate summary: %v", err)
		summary = "Changes implemented as requested."
	}

	if err := o.fork.PushBranch(ctx, workDir, branch, false); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	// Create PR from fork to upstream
	prTitle := fmt.Sprintf("Fix #%d: %s", issue.Number, issue.Title)

	// Build context section based on whether planning was skipped
	var context string
	if issue.SkipPlanning {
		context = "Implemented directly from issue description."
	} else {
		plan, _ := o.db.GetLatestPlan(ctx, issue.ID)
		if plan != nil {
			context = fmt.Sprintf("Based on approved plan v%d.", plan.Version)
		} else {
			context = "Based on approved plan."
		}
	}

	prBody := fmt.Sprintf("## Summary\n\n%s\n\n"+
		"## Context\n\n%s\n\n"+
		"Closes #%d", summary, context, issue.Number)

	// Head is forkOwner:branch, base is the default branch (usually main)
	mention := o.watcher.Mention()
	forkOwner := mention[1:] // Remove @ prefix
	head := fmt.Sprintf("%s:%s", forkOwner, branch)

	pr, err := o.watcher.CreatePR(ctx, prTitle, prBody, head, "main")
	if err != nil {
		return fmt.Errorf("create PR failed: %w", err)
	}

	log.Printf("issue #%d: created PR #%d", issue.Number, pr.Number)

	// Save PR to database
	dbPR := &db.PullRequest{
		IssueID:  issue.ID,
		GitHubID: int64(pr.Number),
		Number:   pr.Number,
		Title:    pr.Title,
		State:    pr.State,
	}
	if err := o.db.CreatePR(ctx, dbPR); err != nil {
		return fmt.Errorf("save PR failed: %w", err)
	}

	// Post completion comment with PR link
	prURL := fmt.Sprintf("https://github.com/%s/%s/pull/%d",
		o.owner, o.repo, pr.Number)
	doneMsg := fmt.Sprintf(
		"🚀 **Implementation complete!**\n\n"+
			"I've created [PR #%d](%s) with the changes.\n\n"+
			"Please review and let me know if any changes are needed.",
		pr.Number, prURL)
	if _, err := o.watcher.PostComment(ctx, issue.Number, doneMsg); err != nil {
		log.Printf("warning: failed to post completion comment: %v", err)
	}

	return o.transitionTo(ctx, issue, state.EventImplementationComplete)
}

func (o *Orchestrator) checkForApproval(
	ctx context.Context, issue *db.Issue,
) error {
	comments, err := o.watcher.FetchIssueComments(ctx, issue.Number)
	if err != nil {
		return fmt.Errorf("fetch comments failed: %w", err)
	}

	filter := commentFilter{
		requireMention: false,
		logPrefix:      fmt.Sprintf("issue #%d", issue.Number),
	}
	newComments, err := o.filterNewMentionComments(ctx, comments, filter)
	if err != nil {
		return err
	}

	if len(newComments) == 0 {
		return nil
	}

	var commentBodies []string
	for _, c := range newComments {
		commentBodies = append(commentBodies, c.Body)
	}

	// TODO: fetch and pass issue labels
	intent, err := o.agent.EvaluateIntent(ctx, issue.Title, issue.Body, nil, commentBodies)
	if err != nil {
		return fmt.Errorf("evaluate intent failed: %w", err)
	}

	for _, ghComment := range newComments {
		if err := o.saveComment(ctx, issue.ID, ghComment); err != nil {
			return fmt.Errorf("save comment failed: %w", err)
		}
	}

	if intent.NeedsClarify {
		if _, err := o.watcher.PostComment(ctx, issue.Number,
			fmt.Sprintf("🤔 %s", intent.Question)); err != nil {
			log.Printf("warning: failed to post clarifying question: %v", err)
		}
		return nil
	}

	if intent.IsQuestion {
		log.Printf("issue #%d: question detected", issue.Number)
		return o.answerQuestion(ctx, issue, intent.Question)
	}

	if intent.IsApproval {
		log.Printf("issue #%d: approval detected", issue.Number)
		return o.ProcessApproval(ctx, issue.ID)
	}

	if intent.IsRevision {
		log.Printf("issue #%d: revision requested: %s", issue.Number, intent.Feedback)
		return o.ProcessRejection(ctx, issue.ID, intent.Feedback)
	}

	return nil
}

func (o *Orchestrator) getCommentBodies(
	ctx context.Context, issueNum int,
) ([]string, error) {
	comments, err := o.watcher.FetchIssueComments(ctx, issueNum)
	if err != nil {
		return nil, err
	}

	agentUsername := o.agentUsername()
	var bodies []string

	for _, c := range comments {
		if c.Author == agentUsername {
			continue
		}
		bodies = append(bodies, c.Body)
	}

	return bodies, nil
}

func (o *Orchestrator) ProcessApproval(
	ctx context.Context, issueID int64,
) error {
	issue, err := o.db.GetIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("get issue failed: %w", err)
	}

	if issue.State != db.StatePlanReview {
		return fmt.Errorf("issue not in plan_review state")
	}

	plan, err := o.db.GetLatestPlan(ctx, issue.ID)
	if err != nil {
		return fmt.Errorf("get plan failed: %w", err)
	}

	if err := o.db.ApprovePlan(ctx, plan.ID); err != nil {
		return fmt.Errorf("approve plan failed: %w", err)
	}

	return o.transitionTo(ctx, issue, state.EventPlanApproved)
}

func (o *Orchestrator) ProcessRejection(
	ctx context.Context, issueID int64, feedback string,
) error {
	issue, err := o.db.GetIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("get issue failed: %w", err)
	}

	if issue.State != db.StatePlanReview {
		return fmt.Errorf("issue not in plan_review state")
	}

	// Get latest plan and store feedback before transitioning
	plan, err := o.db.GetLatestPlan(ctx, issue.ID)
	if err != nil {
		return fmt.Errorf("get plan failed: %w", err)
	}

	if err := o.db.UpdatePlanFeedback(ctx, plan.ID, feedback); err != nil {
		return fmt.Errorf("update plan feedback failed: %w", err)
	}

	return o.transitionTo(ctx, issue, state.EventPlanRejected)
}

func (o *Orchestrator) agentUsername() string {
	return o.watcher.Mention()[1:] // Remove @ prefix
}

type commentFilter struct {
	requireMention bool
	logPrefix      string
}

func (o *Orchestrator) filterNewMentionComments(
	ctx context.Context,
	comments []*watcher.Comment,
	filter commentFilter,
) ([]*watcher.Comment, error) {
	agentUsername := o.agentUsername()
	var newComments []*watcher.Comment

	for _, ghComment := range comments {
		if ghComment.Author == agentUsername {
			continue
		}

		if filter.requireMention && !ghComment.MentionsAgent(o.watcher.Mention()) {
			continue
		}

		existing, err := o.db.GetCommentByGitHubID(ctx, ghComment.ID)
		if err == nil && existing != nil {
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("check comment failed: %w", err)
		}

		newComments = append(newComments, ghComment)
	}

	return newComments, nil
}

func (o *Orchestrator) saveComment(
	ctx context.Context, issueID int64, ghComment *watcher.Comment,
) error {
	dbComment := &db.Comment{
		IssueID:  issueID,
		GitHubID: ghComment.ID,
		Author:   ghComment.Author,
		Body:     ghComment.Body,
		IsOurs:   false,
	}
	return o.db.CreateComment(ctx, dbComment)
}

func (o *Orchestrator) updateContextFile(
	ctx context.Context, issue *db.Issue, workDir string,
) error {
	// Get all comments
	issueComments, err := o.watcher.FetchIssueComments(ctx, issue.Number)
	if err != nil {
		return err
	}

	// Convert to context format
	var comments []*issuecontext.Comment
	for _, c := range issueComments {
		comments = append(comments, &issuecontext.Comment{
			Author: c.Author,
			Body:   c.Body,
		})
	}

	// Get PR comments if in PR state
	var prComments []*issuecontext.PRComment
	if issue.State == db.StatePRCreated || issue.State == db.StatePRReview {
		pr, err := o.db.GetPRByIssueID(ctx, issue.ID)
		if err == nil {
			allPRComments, err := o.watcher.FetchAllPRComments(ctx, pr.Number)
			if err == nil {
				for _, c := range allPRComments {
					prComments = append(prComments, &issuecontext.PRComment{
						Author:   c.Author,
						Body:     c.Body,
						Path:     c.Path,
						Line:     c.Line,
						DiffHunk: c.DiffHunk,
					})
				}
			}
		}
	}

	// Get plan history
	var plans []*issuecontext.Plan
	dbPlans, err := o.db.GetPlanHistory(ctx, issue.ID)
	if err == nil && len(dbPlans) > 0 {
		for _, p := range dbPlans {
			plans = append(plans, &issuecontext.Plan{
				Version:  p.Version,
				Content:  p.Content,
				Feedback: p.Feedback,
				Approved: p.Approved,
			})
		}
	}

	// Build and save
	content := issuecontext.Build(issue, comments, prComments, plans)
	return issuecontext.Save(workDir, content)
}

func (o *Orchestrator) answerQuestion(
	ctx context.Context, issue *db.Issue, question string,
) error {
	return o.answerQuestionOn(ctx, issue, question, issue.Number)
}

func (o *Orchestrator) answerQuestionOn(
	ctx context.Context, issue *db.Issue, question string, targetNum int,
) error {
	// Build context based on issue state
	var issueContext string
	switch issue.State {
	case db.StatePlanReview:
		plan, err := o.db.GetLatestPlan(ctx, issue.ID)
		if err == nil && plan != nil {
			issueContext = fmt.Sprintf("Issue: %s\n\nCurrent plan:\n%s",
				issue.Body, plan.Content)
		} else {
			issueContext = fmt.Sprintf("Issue: %s", issue.Body)
		}
	case db.StatePRReview, db.StatePRCreated:
		issueContext = fmt.Sprintf("Issue: %s\n\nPR is under review.",
			issue.Body)
	default:
		issueContext = fmt.Sprintf("Issue: %s", issue.Body)
	}

	workDir := issue.WorkDir
	if workDir == "" {
		var err error
		workDir, err = o.fork.SetupWorkDir(ctx, o.owner, o.repo, issue.Number)
		if err != nil {
			return fmt.Errorf("setup work dir for question failed: %w", err)
		}
	}

	if err := o.updateContextFile(ctx, issue, workDir); err != nil {
		log.Printf("warning: failed to update context file: %v", err)
	}

	answer, err := o.agent.AnswerQuestion(ctx, workDir, question, issueContext)
	if err != nil {
		return fmt.Errorf("answer question failed: %w", err)
	}

	// Post answer as comment (no state change)
	answerComment := fmt.Sprintf("💬 %s", answer)
	if _, err := o.watcher.PostComment(ctx, targetNum, answerComment); err != nil {
		return fmt.Errorf("post answer failed: %w", err)
	}

	log.Printf("issue #%d: answered question on #%d", issue.Number, targetNum)
	return nil
}

func (o *Orchestrator) processPRComments(
	ctx context.Context, issue *db.Issue,
) error {
	dbPR, err := o.db.GetPRByIssueID(ctx, issue.ID)
	if err != nil {
		return fmt.Errorf("get PR failed: %w", err)
	}

	// Check current PR status from GitHub
	ghPR, err := o.watcher.GetPR(ctx, dbPR.Number)
	if err != nil {
		return fmt.Errorf("fetch PR status failed: %w", err)
	}

	// Handle merged or closed PRs
	if ghPR.Merged {
		log.Printf("PR #%d: merged", dbPR.Number)
		return o.transitionTo(ctx, issue, state.EventPRMerged)
	}
	if ghPR.State == "closed" {
		log.Printf("PR #%d: closed without merge", dbPR.Number)
		return o.transitionTo(ctx, issue, state.EventPRClosed)
	}

	comments, err := o.watcher.FetchAllPRComments(ctx, dbPR.Number)
	if err != nil {
		return fmt.Errorf("fetch PR comments failed: %w", err)
	}

	filter := commentFilter{
		requireMention: true,
		logPrefix:      fmt.Sprintf("PR #%d", dbPR.Number),
	}
	newComments, err := o.filterNewMentionComments(ctx, comments, filter)
	if err != nil {
		return err
	}

	if len(newComments) == 0 {
		return nil
	}

	for _, ghComment := range newComments {
		log.Printf("PR #%d: processing maintainer comment from %s",
			dbPR.Number, ghComment.Author)

		if err := o.saveComment(ctx, issue.ID, ghComment); err != nil {
			return fmt.Errorf("save comment failed: %w", err)
		}

		// Evaluate intent to determine if this is a question or action
		// TODO: fetch and pass issue labels
		intent, err := o.agent.EvaluateIntent(ctx, issue.Title, issue.Body, nil, []string{ghComment.Body})
		if err != nil {
			log.Printf("warning: failed to evaluate PR comment intent: %v", err)
			// Fall back to treating as review comment
			intent = &Intent{}
		}

		if intent.NeedsClarify {
			log.Printf("PR #%d: needs clarification from %s", dbPR.Number, ghComment.Author)
			if _, err := o.watcher.PostComment(ctx, dbPR.Number,
				fmt.Sprintf("🤔 %s", intent.Question)); err != nil {
				log.Printf("warning: failed to post clarifying question: %v", err)
			}
			continue
		}

		if intent.IsQuestion {
			log.Printf("PR #%d: question detected from %s", dbPR.Number, ghComment.Author)
			if err := o.answerQuestionOn(ctx, issue, intent.Question, dbPR.Number); err != nil {
				return fmt.Errorf("answer PR question failed: %w", err)
			}
			continue
		}

		if intent.IsRevision {
			// Handle as code review - make changes
			log.Printf("PR #%d: revision requested from %s", dbPR.Number, ghComment.Author)

			if err := o.updateContextFile(ctx, issue, issue.WorkDir); err != nil {
				log.Printf("warning: failed to update context file: %v", err)
			}

			if err := o.agent.RespondToReview(ctx, issue.WorkDir, ghComment.Context()); err != nil {
				return fmt.Errorf("respond to review failed: %w", err)
			}

			if err := o.fork.PushBranch(ctx, issue.WorkDir, issue.BranchName, false); err != nil {
				return fmt.Errorf("push after review response failed: %w", err)
			}

			ackMsg := fmt.Sprintf(
				"✅ **Addressed review comment from @%s.**\n\n"+
					"I've pushed changes to address this feedback. Please review.",
				ghComment.Author)
			if _, err := o.watcher.PostComment(ctx, dbPR.Number, ackMsg); err != nil {
				log.Printf("warning: failed to post acknowledgment: %v", err)
			}
			continue
		}
	}

	if issue.State == db.StatePRCreated {
		return o.transitionTo(ctx, issue, state.EventPROpened)
	}

	return nil
}
