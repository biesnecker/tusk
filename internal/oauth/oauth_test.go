package oauth

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestGetRandomPort(t *testing.T) {
	port, err := getRandomPort()
	if err != nil {
		t.Fatalf("Failed to get random port: %v", err)
	}

	if port < minPort || port > maxPort {
		t.Errorf("Port %d is outside expected range [%d, %d]", port, minPort, maxPort)
	}
}

func TestNewCallbackServer(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	if cs.port < minPort || cs.port > maxPort {
		t.Errorf("Port %d is outside expected range [%d, %d]", cs.port, minPort, maxPort)
	}

	expectedURI := fmt.Sprintf("http://localhost:%d/callback", cs.port)
	if cs.RedirectURI() != expectedURI {
		t.Errorf("Expected redirect URI %q, got %q", expectedURI, cs.RedirectURI())
	}
}

func TestCallbackServerStart(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/callback?code=test123", cs.port))
	if err != nil {
		t.Fatalf("Failed to make request to callback server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	cs.shutdown()
}

func TestCallbackServerWaitForCode(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}

	expectedCode := "test_auth_code_123"

	go func() {
		time.Sleep(100 * time.Millisecond)
		http.Get(fmt.Sprintf("http://localhost:%d/callback?code=%s", cs.port, expectedCode))
	}()

	code, err := cs.WaitForCode(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to wait for code: %v", err)
	}

	if code != expectedCode {
		t.Errorf("Expected code %q, got %q", expectedCode, code)
	}
}

func TestCallbackServerTimeout(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}

	_, err = cs.WaitForCode(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if err.Error() != "timeout waiting for authorization" {
		t.Errorf("Expected timeout error, got %q", err.Error())
	}
}

func TestCallbackServerMissingCode(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	if err := cs.Start(); err != nil {
		t.Fatalf("Failed to start callback server: %v", err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		http.Get(fmt.Sprintf("http://localhost:%d/callback", cs.port))
	}()

	_, err = cs.WaitForCode(5 * time.Second)
	if err == nil {
		t.Error("Expected error for missing code, got nil")
	}
}

func TestCallbackServerPort(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	port := cs.Port()
	if port != cs.port {
		t.Errorf("Expected port %d, got %d", cs.port, port)
	}
}

func TestCallbackServerRedirectURI(t *testing.T) {
	cs, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("Failed to create callback server: %v", err)
	}

	uri := cs.RedirectURI()
	expected := fmt.Sprintf("http://localhost:%d/callback", cs.port)

	if uri != expected {
		t.Errorf("Expected URI %q, got %q", expected, uri)
	}
}
