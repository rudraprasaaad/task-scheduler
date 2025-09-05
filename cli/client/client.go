package client

import (
	"bytes"
	"encoding/json"
	"fmt"
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
