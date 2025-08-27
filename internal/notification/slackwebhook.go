package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	body := SlackWebhookMessage{
		Text: msg,
	}
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
		return fmt.Errorf("failed to send slack webhook request: %w", err)
	}
	return nil
}

func (s *SlackWebhook) dirLink(dir string) string {
	if s == nil || s.Repo == "" {
		return dir
	}
	q := "is%3Apr+is%3Aopen+label%3A" + url.QueryEscape(dir)
	u := fmt.Sprintf("https://github.com/%s/pulls?q=%s", s.Repo, q)
	return fmt.Sprintf("<%s|%s>", u, dir)
}

func workspaceLine(workspace string) string {
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	return fmt.Sprintf("\n*Workspace*: `%s`", workspace)
}

func (s *SlackWebhook) ExtraWorkspaceInRemote(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, fmt.Sprintf("⚠️ *Extra workspace in remote*\n*Directory*: %s%s", s.dirLink(dir), workspaceLine(workspace)))
}

func (s *SlackWebhook) MissingWorkspaceInRemote(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, fmt.Sprintf("⚠️ *Missing workspace in remote*\n*Directory*: %s%s", s.dirLink(dir), workspaceLine(workspace)))
}

func (s *SlackWebhook) PlanDrift(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, fmt.Sprintf("🚨 *Drift detected*\n*Directory*: %s%s", s.dirLink(dir), workspaceLine(workspace)))
}

func (s *SlackWebhook) PlanFailed(ctx context.Context, dir string, workspace string) error {
	return s.sendSlackMessage(ctx, fmt.Sprintf("🔥 *Drift plan failed*\n*Directory*: %s%s", s.dirLink(dir), workspaceLine(workspace)))
}

var _ Notification = &SlackWebhook{}
