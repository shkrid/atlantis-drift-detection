package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackWebhook struct {
	WebhookURL string
	HTTPClient *http.Client
	Repo       string
}

func NewSlackWebhook(webhookURL string, HTTPClient *http.Client, repo string) *SlackWebhook {
	if webhookURL == "" {
		return nil
	}
	return &SlackWebhook{
		WebhookURL: webhookURL,
		HTTPClient: HTTPClient,
		Repo:       repo,
	}
}

type SlackWebhookMessage struct {
	Text string `json:"text"`
}

func (s *SlackWebhook) sendSlackMessage(ctx context.Context, msg string) error {
	body := SlackWebhookMessage{Text: msg}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal slack webhook message: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create slack webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack webhook request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send slack webhook request: %s", resp.Status)
	}
	return nil
}

func (s *SlackWebhook) ExtraWorkspaceInRemote(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, BuildSlackText("⚠️", "Extra workspace in remote", dir, workspace, s.Repo))
}

func (s *SlackWebhook) MissingWorkspaceInRemote(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, BuildSlackText("⚠️", "Missing workspace in remote", dir, workspace, s.Repo))
}

func (s *SlackWebhook) PlanDrift(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, BuildSlackText("🚨", "Plan Drift", dir, workspace, s.Repo))
}

func (s *SlackWebhook) PlanFailed(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, BuildSlackText("🔥", "Drift plan failed", dir, workspace, s.Repo))
}

var _ Notification = &SlackWebhook{}
