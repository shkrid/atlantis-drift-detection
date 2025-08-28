package atlantis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/runatlantis/atlantis/server/controllers"
	"github.com/runatlantis/atlantis/server/events/models"
	"go.uber.org/zap"
)

type Client struct {
	AtlantisHostname string
	Token            string
	HTTPClient       *http.Client
	Logger           *zap.Logger
}

type PlanSummaryRequest struct {
	Repo      string
	Ref       string
	Type      string
	Dir       string
	Workspace string
}

type PlanResult struct {
	Summaries []PlanSummary
}

type PlanSummary struct {
	HasError bool
	HasLock bool
	Summary string
}

func (p *PlanResult) HasChanges() bool {
	for _, summary := range p.Summaries {
		if summary.HasLock {
			continue
		}
		if summary.HasError {
			continue
		}
		if !strings.Contains(summary.Summary, "No changes. ") {
			return true
		}
	}
	return false
}

func (p *PlanResult) IsLocked() bool {
	for _, summary := range p.Summaries {
		if !summary.HasLock {
			return false
		}
	}
	return true
}

func (p *PlanResult) IsFailed() bool {
	for _, summary := range p.Summaries {
		if summary.HasError {
			return true
		}
	}
	return false
}

type resultResponse struct {
    Error          any
    Failure        string
    ProjectResults []projectResult
}

type projectResult struct {
    RepoRelDir   string
    Workspace    string
    Error        any
    Failure      string
    PlanSuccess  *models.PlanSuccess
}

type ctxKey string

const WorkerIndexKey ctxKey = "worker-index"

func prFromWorkerIndex(ctx context.Context) int {
	if idx, ok := ctx.Value(WorkerIndexKey).(int); ok && idx > 0 {
		return -idx
	}
	return 0
}

func (c *Client) PlanSummary(ctx context.Context, req *PlanSummaryRequest) (*PlanResult, error) {
	planBody := controllers.APIRequest{
		Repository: req.Repo,
		Ref:        req.Ref,
		Type:       req.Type,
		PR:         prFromWorkerIndex(ctx),
		Paths: []struct {
			Directory string
			Workspace string
		}{
			{
				Directory: req.Dir,
				Workspace: req.Workspace,
			},
		},
	}
	planBodyJSON, err := json.Marshal(planBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling plan body: %w", err)
	}
	destination := fmt.Sprintf("%s/api/plan", c.AtlantisHostname)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, destination, strings.NewReader(string(planBodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("error parsing destination: %w", err)
	}
	httpReq.Header.Set("X-Atlantis-Token", c.Token)
	httpReq = httpReq.WithContext(ctx)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error making plan request to %s: %w", destination, err)
	}

	var fullBody bytes.Buffer
	if _, err := io.Copy(&fullBody, resp.Body); err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("unable to close response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		return nil, fmt.Errorf("non-200 and non-500 response for %s: (code:%d)(body:%s)", destination, resp.StatusCode, fullBody.String())
	}

	var result resultResponse
	if err := json.NewDecoder(bytes.NewReader(fullBody.Bytes())).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding plan response(code:%d)(body:%s): %w", resp.StatusCode, fullBody.String(), err)
	}
	c.Logger.Debug("plan result", zap.Any("result", result))

	if result.Error != nil {
		return nil, fmt.Errorf("error making plan request: %v", result.Error)
	}

	if result.Failure != "" {
		return nil, fmt.Errorf("failure making plan request: %s", result.Failure)
	}

	var ret PlanResult
	for _, result := range result.ProjectResults {
		if result.Failure != "" {
			if strings.Contains(result.Failure, "This project is currently locked ") {
				ret.Summaries = append(ret.Summaries, PlanSummary{HasLock: true})
				continue
			}
		}
		if result.PlanSuccess != nil {
			summary := result.PlanSuccess.Summary()
			ret.Summaries = append(ret.Summaries, PlanSummary{Summary: summary})
			continue
		}
		if result.Error != nil {
			ret.Summaries = append(ret.Summaries, PlanSummary{HasError: true})
			continue
		}
		return nil, fmt.Errorf("project result unknown failure: %s", result.Failure)

	}
	return &ret, nil
}
