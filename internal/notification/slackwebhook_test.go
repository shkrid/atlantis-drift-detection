package notification

import (
	"net/http"
	"testing"

	"github.com/cresta/atlantis-drift-detection/internal/testhelper"
)

func TestSlackWebhook_ExtraWorkspaceInRemote(t *testing.T) {
	testhelper.ReadEnvFile(t, "../../")
	wh := NewSlackWebhook(testhelper.EnvOrSkip(t, "SLACK_WEBHOOK_URL"), http.DefaultClient, testhelper.EnvOrSkip(t, "REPO"))
	genericNotificationTest(t, wh)
}
