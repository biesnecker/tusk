package mastodon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Client struct {
	BaseURL     string
	AccessToken string
	HTTPClient  *http.Client
}

type App struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type Status struct {
	ID        string `json:"id"`
	URI       string `json:"uri"`
	URL       string `json:"url"`
	Content   string `json:"content"`
	InReplyTo string `json:"in_reply_to_id"`
}

type MediaAttachment struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	PreviewURL  string `json:"preview_url"`
	Description string `json:"description"`
}

type StatusParams struct {
	Status      string
	InReplyToID string
	Visibility  string
	SpoilerText string
	MediaIDs    []string
}

func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		HTTPClient:  &http.Client{},
	}
}

func (c *Client) RegisterApp(appName, redirectURI, scopes string) (*App, error) {
	endpoint := fmt.Sprintf("%s/api/v1/apps", c.BaseURL)

	data := url.Values{}
	data.Set("client_name", appName)
	data.Set("redirect_uris", redirectURI)
	data.Set("scopes", scopes)

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to register app: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to register app: %s (status %d)", string(body), resp.StatusCode)
	}

	var app App
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode app response: %w", err)
	}

	return &app, nil
}

func (c *Client) GetAuthorizationURL(clientID, redirectURI, scopes string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", scopes)

	return fmt.Sprintf("%s/oauth/authorize?%s", c.BaseURL, params.Encode())
}

func (c *Client) GetAccessToken(clientID, clientSecret, redirectURI, code string) (string, error) {
	endpoint := fmt.Sprintf("%s/oauth/token", c.BaseURL)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get access token: %s (status %d)", string(body), resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return result.AccessToken, nil
}

func (c *Client) PostStatus(params StatusParams) (*Status, error) {
	endpoint := fmt.Sprintf("%s/api/v1/statuses", c.BaseURL)

	payload := map[string]interface{}{
		"status": params.Status,
	}

	if params.InReplyToID != "" {
		payload["in_reply_to_id"] = params.InReplyToID
	}

	if params.Visibility != "" {
		payload["visibility"] = params.Visibility
	}

	if params.SpoilerText != "" {
		payload["spoiler_text"] = params.SpoilerText
	}

	if len(params.MediaIDs) > 0 {
		payload["media_ids"] = params.MediaIDs
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to post status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to post status: %s (status %d)", string(body), resp.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

func (c *Client) GetStatus(id string) (*Status, error) {
	endpoint := fmt.Sprintf("%s/api/v1/statuses/%s", c.BaseURL, id)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get status: %s (status %d)", string(body), resp.StatusCode)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

func (c *Client) DeleteStatus(id string) error {
	endpoint := fmt.Sprintf("%s/api/v1/statuses/%s", c.BaseURL, id)

	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete status: %s (status %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (c *Client) UploadMedia(fileData []byte, filename, mimeType, description string) (*MediaAttachment, error) {
	endpoint := fmt.Sprintf("%s/api/v2/media", c.BaseURL)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Add description if provided
	if description != "" {
		if err := writer.WriteField("description", description); err != nil {
			return nil, fmt.Errorf("failed to write description field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to upload media: %s (status %d)", string(body), resp.StatusCode)
	}

	var media MediaAttachment
	if err := json.NewDecoder(resp.Body).Decode(&media); err != nil {
		return nil, fmt.Errorf("failed to decode media response: %w", err)
	}

	return &media, nil
}

func (c *Client) RevokeToken(clientID, clientSecret string) error {
	endpoint := fmt.Sprintf("%s/oauth/revoke", c.BaseURL)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("token", c.AccessToken)

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to revoke token: %s (status %d)", string(body), resp.StatusCode)
	}

	return nil
}
