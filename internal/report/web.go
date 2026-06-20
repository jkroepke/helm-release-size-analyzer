package report

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cli/browser"
	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
)

const (
	webDataPlaceholder    = "__HELM_RELEASE_SIZE_ANALYZER_DATA__"
	webNoncePlaceholder   = "__HELM_RELEASE_SIZE_ANALYZER_NONCE__"
	webVersionPlaceholder = "__HELM_RELEASE_SIZE_ANALYZER_VERSION__"
	shutdownTimeout       = 5 * time.Second
	nonceBytes            = 18
)

//go:embed web.html
var webPage string

// ServeWeb serves an interactive report on a random loopback port until the
// context is canceled or the user stops the server from the page.
func ServeWeb(ctx context.Context, tree analyze.Tree, version string, ready func(string)) error {
	page, nonce, err := renderWebPage(tree, version)
	if err != nil {
		return err
	}

	listenerConfig := net.ListenConfig{}

	listener, err := listenerConfig.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen for web report: %w", err)
	}

	stop := make(chan struct{}, 1)
	server := newWebServer(page, nonce, stop)
	serveErrors := make(chan error, 1)

	go func() {
		serveErrors <- server.Serve(listener)
	}()

	ready("http://" + listener.Addr().String())

	select {
	case <-ctx.Done():
	case <-stop:
	case err = <-serveErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve web report: %w", err)
		}

		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	if err != nil {
		return fmt.Errorf("shut down web report: %w", err)
	}

	return nil
}

// OpenBrowser opens url in the operating system's default browser.
func OpenBrowser(url string) error {
	err := browser.OpenURL(url)
	if err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	return nil
}

func renderWebPage(tree analyze.Tree, version string) (string, string, error) {
	data, err := json.Marshal(tree)
	if err != nil {
		return "", "", fmt.Errorf("encode web report: %w", err)
	}

	nonce, err := generateNonce()
	if err != nil {
		return "", "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	page := strings.Replace(webPage, webDataPlaceholder, encoded, 1)
	page = strings.ReplaceAll(page, webNoncePlaceholder, nonce)
	page = strings.Replace(page, webVersionPlaceholder, html.EscapeString(prefixedVersion(version)), 1)

	return page, nonce, nil
}

func prefixedVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}

	return "v" + version
}

func generateNonce() (string, error) {
	data := make([]byte, nonceBytes)

	_, err := rand.Read(data)
	if err != nil {
		return "", fmt.Errorf("generate CSP nonce: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(data), nil
}

func newWebServer(page, nonce string, stop chan<- struct{}) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(response http.ResponseWriter, _ *http.Request) {
		setWebHeaders(response, nonce)
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = response.Write([]byte(page))
	})
	mux.HandleFunc("POST /shutdown", func(response http.ResponseWriter, _ *http.Request) {
		setWebHeaders(response, nonce)
		response.WriteHeader(http.StatusNoContent)

		select {
		case stop <- struct{}{}:
		default:
		}
	})

	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func setWebHeaders(response http.ResponseWriter, nonce string) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Security-Policy", contentSecurityPolicy(nonce))
	response.Header().Set("X-Content-Type-Options", "nosniff")
}

func contentSecurityPolicy(nonce string) string {
	return fmt.Sprintf(
		"default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-%s'; connect-src 'self'",
		nonce,
	)
}
