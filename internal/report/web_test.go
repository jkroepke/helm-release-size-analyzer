package report_test

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
	"github.com/jkroepke/helm-release-size-analyzer/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeWebRendersTreeAndStops(t *testing.T) {
	t.Parallel()

	tree := analyze.Tree{Root: analyze.TreeNode{
		Name:  "root",
		Kind:  "object",
		Bytes: 21,
		Children: []analyze.TreeNode{
			{Name: "message", Kind: "string", Preview: "hello", Bytes: 19},
		},
	}}
	result := make(chan webResult, 1)
	client := &http.Client{Timeout: 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := report.ServeWeb(ctx, tree, func(url string) {
		go fetchWebReport(client, url, result)
	})
	require.NoError(t, err)

	web := <-result
	assert.Contains(t, web.page, "Helm release size analyzer")
	assert.NotContains(t, web.page, "__HELM_RELEASE_SIZE_ANALYZER_DATA__")
	assert.NotContains(t, web.page, "__HELM_RELEASE_SIZE_ANALYZER_NONCE__")

	nonceMatch := regexp.MustCompile(`<script nonce="([^"]+)">`).FindStringSubmatch(web.page)
	require.Len(t, nonceMatch, 2)
	assert.Contains(t, web.page, `script-src 'nonce-`+nonceMatch[1]+`'`)
	assert.Contains(t, web.csp, `script-src 'nonce-`+nonceMatch[1]+`'`)
	assert.NotContains(t, web.csp, "script-src 'unsafe-inline'")
}

type webResult struct {
	page string
	csp  string
}

func fetchWebReport(client *http.Client, url string, result chan<- webResult) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		result <- webResult{page: err.Error()}

		return
	}

	response, err := client.Do(request)
	if err != nil {
		result <- webResult{page: err.Error()}

		return
	}

	page, err := io.ReadAll(response.Body)
	_ = response.Body.Close()

	if err != nil {
		result <- webResult{page: err.Error()}

		return
	}

	request, err = http.NewRequestWithContext(context.Background(), http.MethodPost, url+"/shutdown", strings.NewReader(""))
	if err == nil {
		shutdownResponse, requestErr := client.Do(request)
		if requestErr == nil {
			_ = shutdownResponse.Body.Close()
		}
	}

	result <- webResult{page: string(page), csp: response.Header.Get("Content-Security-Policy")}
}
