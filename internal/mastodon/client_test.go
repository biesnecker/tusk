package mastodon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://mastodon.social"
	token := "test_token"

	client := NewClient(baseURL, token)

	if client.BaseURL != baseURL {
		t.Errorf("Expected BaseURL %q, got %q", baseURL, client.BaseURL)
	}

	if client.AccessToken != token {
		t.Errorf("Expected AccessToken %q, got %q", token, client.AccessToken)
	}

	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}
}

func TestRegisterApp(t *testing.T) {
	expectedApp := &App{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/apps" {
			t.Errorf("Expected path /api/v1/apps, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("client_name") != "TestApp" {
			t.Errorf("Expected client_name TestApp, got %s", r.FormValue("client_name"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedApp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	app, err := client.RegisterApp("TestApp", "http://localhost:8080/callback", "read write")

	if err != nil {
		t.Fatalf("Failed to register app: %v", err)
	}

	if app.ClientID != expectedApp.ClientID {
		t.Errorf("Expected ClientID %q, got %q", expectedApp.ClientID, app.ClientID)
	}

	if app.ClientSecret != expectedApp.ClientSecret {
		t.Errorf("Expected ClientSecret %q, got %q", expectedApp.ClientSecret, app.ClientSecret)
	}
}

func TestGetAuthorizationURL(t *testing.T) {
	client := NewClient("https://mastodon.social", "")
	clientID := "test_client_id"
	redirectURI := "http://localhost:8080/callback"
	scopes := "read write"

	authURL := client.GetAuthorizationURL(clientID, redirectURI, scopes)

	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Failed to parse auth URL: %v", err)
	}

	if parsedURL.Scheme != "https" {
		t.Errorf("Expected scheme https, got %s", parsedURL.Scheme)
	}

	if parsedURL.Host != "mastodon.social" {
		t.Errorf("Expected host mastodon.social, got %s", parsedURL.Host)
	}

	query := parsedURL.Query()
	if query.Get("client_id") != clientID {
		t.Errorf("Expected client_id %q, got %q", clientID, query.Get("client_id"))
	}

	if query.Get("redirect_uri") != redirectURI {
		t.Errorf("Expected redirect_uri %q, got %q", redirectURI, query.Get("redirect_uri"))
	}

	if query.Get("response_type") != "code" {
		t.Errorf("Expected response_type code, got %s", query.Get("response_type"))
	}
}

func TestGetAccessToken(t *testing.T) {
	expectedToken := "test_access_token_xyz"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("Expected path /oauth/token, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		response := map[string]string{
			"access_token": expectedToken,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	token, err := client.GetAccessToken("client_id", "client_secret", "redirect_uri", "auth_code")

	if err != nil {
		t.Fatalf("Failed to get access token: %v", err)
	}

	if token != expectedToken {
		t.Errorf("Expected token %q, got %q", expectedToken, token)
	}
}

func TestPostStatus(t *testing.T) {
	expectedStatus := &Status{
		ID:      "123456",
		URL:     "https://mastodon.social/@user/123456",
		Content: "Test status",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/statuses" {
			t.Errorf("Expected path /api/v1/statuses, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_token" {
			t.Errorf("Expected Authorization header 'Bearer test_token', got %q", authHeader)
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if payload["status"] != "Test status" {
			t.Errorf("Expected status 'Test status', got %v", payload["status"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedStatus)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_token")
	status, err := client.PostStatus(StatusParams{
		Status: "Test status",
	})

	if err != nil {
		t.Fatalf("Failed to post status: %v", err)
	}

	if status.ID != expectedStatus.ID {
		t.Errorf("Expected ID %q, got %q", expectedStatus.ID, status.ID)
	}

	if status.URL != expectedStatus.URL {
		t.Errorf("Expected URL %q, got %q", expectedStatus.URL, status.URL)
	}
}

func TestPostStatusWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if payload["in_reply_to_id"] != "999" {
			t.Errorf("Expected in_reply_to_id '999', got %v", payload["in_reply_to_id"])
		}

		if payload["visibility"] != "unlisted" {
			t.Errorf("Expected visibility 'unlisted', got %v", payload["visibility"])
		}

		if payload["spoiler_text"] != "CW: test" {
			t.Errorf("Expected spoiler_text 'CW: test', got %v", payload["spoiler_text"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&Status{ID: "123"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_token")
	_, err := client.PostStatus(StatusParams{
		Status:      "Test",
		InReplyToID: "999",
		Visibility:  "unlisted",
		SpoilerText: "CW: test",
	})

	if err != nil {
		t.Fatalf("Failed to post status: %v", err)
	}
}

func TestGetStatus(t *testing.T) {
	expectedStatus := &Status{
		ID:      "123456",
		URL:     "https://mastodon.social/@user/123456",
		Content: "Test status",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/statuses/123456" {
			t.Errorf("Expected path /api/v1/statuses/123456, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_token" {
			t.Errorf("Expected Authorization header 'Bearer test_token', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedStatus)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_token")
	status, err := client.GetStatus("123456")

	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.ID != expectedStatus.ID {
		t.Errorf("Expected ID %q, got %q", expectedStatus.ID, status.ID)
	}
}

func TestDeleteStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/statuses/123456" {
			t.Errorf("Expected path /api/v1/statuses/123456, got %s", r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_token" {
			t.Errorf("Expected Authorization header 'Bearer test_token', got %q", authHeader)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_token")
	err := client.DeleteStatus("123456")

	if err != nil {
		t.Fatalf("Failed to delete status: %v", err)
	}
}

func TestRevokeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/revoke" {
			t.Errorf("Expected path /oauth/revoke, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("Failed to parse form: %v", err)
		}

		if r.FormValue("token") != "test_token" {
			t.Errorf("Expected token 'test_token', got %s", r.FormValue("token"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_token")
	err := client.RevokeToken("client_id", "client_secret")

	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}
}
