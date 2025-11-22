package oauth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

const (
	minPort = 49152
	maxPort = 65535
)

type CallbackServer struct {
	port     int
	server   *http.Server
	codeChan chan string
	errChan  chan error
	once     sync.Once
}

func getRandomPort() (int, error) {
	portRange := maxPort - minPort + 1
	n, err := rand.Int(rand.Reader, big.NewInt(int64(portRange)))
	if err != nil {
		return 0, err
	}
	return minPort + int(n.Int64()), nil
}

func NewCallbackServer() (*CallbackServer, error) {
	port, err := getRandomPort()
	if err != nil {
		return nil, fmt.Errorf("failed to get random port: %w", err)
	}

	cs := &CallbackServer{
		port:     port,
		codeChan: make(chan string, 1),
		errChan:  make(chan error, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", cs.handleCallback)

	cs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return cs, nil
}

func (cs *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		cs.errChan <- fmt.Errorf("no authorization code received")
		http.Error(w, "Authorization failed: no code received", http.StatusBadRequest)
		return
	}

	cs.codeChan <- code

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Tusk Authorization</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #f5f5f5;
        }
        .container {
            text-align: center;
            background: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 { color: #2ecc71; }
    </style>
</head>
<body>
    <div class="container">
        <h1>&#128640; Authorization Successful!</h1>
        <p>You can close this window and return to your terminal.</p>
    </div>
</body>
</html>
`)
}

func (cs *CallbackServer) Start() error {
	listener, err := net.Listen("tcp", cs.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	go func() {
		if err := cs.server.Serve(listener); err != http.ErrServerClosed {
			cs.errChan <- err
		}
	}()

	return nil
}

func (cs *CallbackServer) WaitForCode(timeout time.Duration) (string, error) {
	select {
	case code := <-cs.codeChan:
		cs.shutdown()
		return code, nil
	case err := <-cs.errChan:
		cs.shutdown()
		return "", err
	case <-time.After(timeout):
		cs.shutdown()
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

func (cs *CallbackServer) shutdown() {
	cs.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cs.server.Shutdown(ctx)
	})
}

func (cs *CallbackServer) Port() int {
	return cs.port
}

func (cs *CallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://localhost:%d/callback", cs.port)
}

func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
