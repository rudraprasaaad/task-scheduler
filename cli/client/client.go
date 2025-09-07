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
	"sync"
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

type TaskDetail struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
}

type QueueStatus struct {
	QueueSize int       `json:"queue_size"`
	Timestamp time.Time `json:"timestamp"`
}

type WorkerStats struct {
	ID       string    `json:"id"`
	Status   string    `json:"status"`
	TasksRun int       `json:"tasks_run"`
	LastSeen time.Time `json:"last_seen"`
}

type SystemStatus struct {
	QueueSize int
	Workers   []WorkerStats
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

func (c *Client) GetTask(taskID string) (*TaskDetail, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.BaseURL, taskID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send get task request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("get task failed with status %s: %s", res.Status, string(body))
	}

	var taskDetail TaskDetail
	if err := json.NewDecoder(res.Body).Decode(&taskDetail); err != nil {
		return nil, fmt.Errorf("failed to decode get task response: %w", err)
	}

	return &taskDetail, nil
}

func (c *Client) CancelTaskk(taskID string) error {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/cancel", c.BaseURL, taskID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	res, err := c.HTTPClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send cancel task request: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("cancel task failed with status %s: %s", res.Status, string(body))
	}

	return nil
}

func (c *Client) GetSystemStatus() (*SystemStatus, error) {
	var queueStatus QueueStatus
	var workerStats struct {
		Workers []WorkerStats `json:"workers"`
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		url := fmt.Sprintf("%s/api/v1/queue/status", c.BaseURL)
		req, err := http.NewRequest("GET", url, nil)

		if err != nil {
			errs <- err
			return
		}

		res, err := c.HTTPClient.Do(req)
		if err != nil {
			errs <- err
			return
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			errs <- fmt.Errorf("queue status failed to with status: %s", res.Status)
			return
		}

		if err := json.NewDecoder(res.Body).Decode(&queueStatus); err != nil {
			errs <- err
		}
	}()

	go func() {
		defer wg.Done()
		url := fmt.Sprintf("%s/api/v1/workers/stats", c.BaseURL)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			errs <- err
			return
		}
		res, err := c.HTTPClient.Do(req)
		if err != nil {
			errs <- err
			return
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			errs <- fmt.Errorf("worker stats failed with stats: %s", res.Status)
			return
		}
		if err := json.NewDecoder(res.Body).Decode(&workerStats); err != nil {
			errs <- err
		}
	}()

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	return &SystemStatus{
		QueueSize: queueStatus.QueueSize,
		Workers:   workerStats.Workers,
	}, nil
}
