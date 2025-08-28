package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackAPI struct {
	Token      string
	Channel    string
	HTTPClient *http.Client
	Repo       string
}

type slackPostMessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type slackPostMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func NewSlackAPI(token, channel, repo string, httpClient *http.Client) *SlackAPI {
	if token == "" || channel == "" {
		return nil
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &SlackAPI{Token: token, Channel: channel, Repo: repo, HTTPClient: httpClient}
}

func (s *SlackAPI) post(ctx context.Context, text string) error {
	body, _ := json.Marshal(slackPostMessageRequest{Channel: s.Channel, Text: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post slack message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			// simple single retry after delay
			t, _ := time.ParseDuration(ra + "s")
			time.Sleep(t)
			return s.post(ctx, text)
		}
	}
	var out slackPostMessageResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK || !out.OK {
		return fmt.Errorf("slack api error: status=%d ok=%v err=%s", resp.StatusCode, out.OK, out.Error)
	}
	return nil
}

func (s *SlackAPI) ExtraWorkspaceInRemote(ctx context.Context, dir string, workspace string) error {
	return s.post(ctx, BuildSlackText("⚠️", "Extra workspace in remote", dir, workspace, s.Repo))
}

func (s *SlackAPI) MissingWorkspaceInRemote(ctx context.Context, dir string, workspace string) error {
	return s.post(ctx, BuildSlackText("⚠️", "Missing workspace in remote", dir, workspace, s.Repo))
}

func (s *SlackAPI) PlanDrift(ctx context.Context, dir string, workspace string) error {
	return s.post(ctx, BuildSlackText("🚨", "Plan Drift", dir, workspace, s.Repo))
}

func (s *SlackAPI) PlanFailed(ctx context.Context, dir string, workspace string) error {
	return s.post(ctx, BuildSlackText("🔥", "Drift plan failed", dir, workspace, s.Repo))
}

var _ Notification = &SlackAPI{}
