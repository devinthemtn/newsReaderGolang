package raindrop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/thomaskoefod/newsreadr/pkg/models"
)

const raindropAPIURL = "https://api.raindrop.io/rest/v1"

type Client struct {
	apiToken string
	client   *http.Client
}

type RaindropItem struct {
	Link  string `json:"link"`
	Title string `json:"title"`
	Excerpt string `json:"excerpt,omitempty"`
}

type RaindropResponse struct {
	Result bool   `json:"result"`
	Item   *RaindropItem `json:"item,omitempty"`
}

func NewClient(apiToken string) *Client {
	return &Client{
		apiToken: apiToken,
		client:   &http.Client{},
	}
}

// SaveArticle saves an article to Raindrop.io
func (c *Client) SaveArticle(article *models.Article) error {
	item := RaindropItem{
		Link:    article.URL,
		Title:   article.Title,
		Excerpt: article.Description,
	}

	jsonData, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshaling article: %w", err)
	}

	url := fmt.Sprintf("%s/raindrop", raindropAPIURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to Raindrop: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Raindrop API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result RaindropResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if !result.Result {
		return fmt.Errorf("Raindrop API returned failure")
	}

	return nil
}

// TestConnection tests the API token by making a simple request
func (c *Client) TestConnection() error {
	url := fmt.Sprintf("%s/user", raindropAPIURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to Raindrop: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Raindrop API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
