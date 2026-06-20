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

	tree := analyze.Tree{CompressedBytes: 12, Root: analyze.TreeNode{
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

	err := report.ServeWeb(ctx, tree, "0.0.0", func(url string) {
		go fetchWebReport(client, url, result)
	})
	require.NoError(t, err)

	web := <-result
	assert.Contains(t, web.page, "Helm release size analyzer")
	assert.NotContains(t, web.page, "__HELM_RELEASE_SIZE_ANALYZER_DATA__")
	assert.NotContains(t, web.page, "__HELM_RELEASE_SIZE_ANALYZER_NONCE__")
	assert.NotContains(t, web.page, "__HELM_RELEASE_SIZE_ANALYZER_VERSION__")
	assert.Contains(t, web.page, `href="https://github.com/jkroepke/helm-release-size-analyzer"`)
	assert.Contains(t, web.page, ">helm-release-size-analyzer</a> v0.0.0<br>by Jan-Otto Kröpke")

	nonceMatch := regexp.MustCompile(`<script nonce="([^"]+)">`).FindStringSubmatch(web.page)
	require.Len(t, nonceMatch, 2)
	assert.Contains(t, web.page, `<meta http-equiv="Content-Security-Policy"`)
	assert.Contains(t, web.page, `script-src 'nonce-`+nonceMatch[1]+`'`)
	assert.Contains(t, web.csp, `script-src 'nonce-`+nonceMatch[1]+`'`)
	assert.NotContains(t, web.csp, "script-src 'unsafe-inline'")
	assert.Contains(t, web.page, "const childNodes = [...node.children].sort(bySize)")
	assert.Contains(t, web.page, "summary.style.setProperty('--size-percent', percentage + '%')")
	assert.Contains(t, web.page, `aria-label="Release size by property"`)
	assert.Contains(t, web.page, `role="status" aria-live="polite"`)
	assert.Contains(t, web.page, "button:focus-visible, summary:focus-visible")
	assert.Contains(t, web.page, "@media (prefers-color-scheme: dark)")
	assert.Contains(t, web.page, "Compressed Secret value")
	assert.Contains(t, web.page, "bytes < 1024 ? bytes.toLocaleString() + ' B' : (bytes / 1024).toFixed(2) + ' KB'")
	assert.Contains(t, web.page, "node.name + ' (' + node.label + ')'")
	assert.Contains(t, web.page, "1,048,576-byte Secret data limit")
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
