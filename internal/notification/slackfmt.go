package notification

import (
	"fmt"
	"net/url"
)

func BuildSlackText(emoji, title, dir, workspace, repo string) string {
	text := fmt.Sprintf("%s *%s*\n", emoji, title)
	dirLine := dir
	if dir != "" && repo != "" {
		encoded := url.QueryEscape(dir)
		linkURL := fmt.Sprintf("https://github.com/%s/pulls?q=is%%3Apr+is%%3Aopen+label%%3A%s", repo, encoded)
		dirLine = fmt.Sprintf("<%s|%s>", linkURL, dir)
	}
	if dirLine != "" {
		text += fmt.Sprintf("*Directory*: %s\n", dirLine)
	}
	if workspace != "" {
		text += fmt.Sprintf("*Workspace*: %s\n", workspace)
	}
	return text
}
