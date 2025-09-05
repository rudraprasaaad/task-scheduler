package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Task struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTaskPayload struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
	Priority int             `json:"priority"`
}

func (c *Client) CreateTask(payload CreateTaskPayload) (*Task, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create task payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/tasks", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send create task request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("create task failed with status %s: %s", res.Status, string(body))
	}

	var createdTask Task
	if err := json.NewDecoder(res.Body).Decode(&createdTask); err != nil {
		return nil, fmt.Errorf("failed to decode create task response: %w", err)
	}

	return &createdTask, nil
}

func NewClient(baseURL string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Jar:     jar,
			Timeout: 1 * time.Minute,
		},
	}

	if err := client.loadCookies(); err != nil {
		fmt.Fprintln(os.Stderr, "No saved session found. Please login.")
	}

	return client, nil
}

func (c *Client) Login(email, password string) error {
	payload, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/auth/login", bytes.NewReader(payload))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	return c.saveCookies()
}

func (c *Client) getCookieFile() (string, error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".task-cli-cookies"), nil
}

func (c *Client) saveCookies() error {
	cookieFile, err := c.getCookieFile()
	if err != nil {
		return err
	}

	f, err := os.Create(cookieFile)
	if err != nil {
		return err
	}
	defer f.Close()

	apiURL, _ := url.Parse(c.BaseURL)
	cookies := c.HTTPClient.Jar.Cookies(apiURL)

	return json.NewEncoder(f).Encode(cookies)
}

func (c *Client) loadCookies() error {
	cookieFile, err := c.getCookieFile()
	if err != nil {
		return err
	}

	f, err := os.Open(cookieFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var cookies []*http.Cookie
	if err := json.NewDecoder(f).Decode(&cookies); err != nil {
		return err
	}

	apiURL, _ := url.Parse(c.BaseURL)
	c.HTTPClient.Jar.SetCookies(apiURL, cookies)
	return nil
}

func (c *Client) ListTasks(limit, offset int) ([]Task, error) {
	url := fmt.Sprintf("%s/api/v1/tasks?limit=%d&offset=%d", c.BaseURL, limit, offset)
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("failed to send list tasks request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tasks failed with status: %s", res.Status)
	}

	var response struct {
		Tasks []Task `json:"tasks"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decoee list tasks response: %w", err)
	}

	return response.Tasks, nil
}
