package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/thomaskoefod/newsreadr/internal/database"
	"github.com/thomaskoefod/newsreadr/pkg/models"
)

type Client struct {
	host   string
	model  string
	db     *database.DB
	client *http.Client
}

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func NewClient(host, model string, db *database.DB) *Client {
	return &Client{
		host:   host,
		model:  model,
		db:     db,
		client: &http.Client{},
	}
}

// GetEmbedding generates an embedding for the given text
func (c *Client) GetEmbedding(text string) ([]float64, error) {
	reqBody := EmbeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", c.host)
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("sending request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return embResp.Embedding, nil
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ScoreArticle calculates relevance score for an article based on user interests
func (c *Client) ScoreArticle(article *models.Article, interests []models.UserInterest) (float64, error) {
	// Create text representation of article for embedding
	articleText := fmt.Sprintf("%s. %s", article.Title, article.Description)

	// Get article embedding
	articleEmb, err := c.GetEmbedding(articleText)
	if err != nil {
		return 0, fmt.Errorf("getting article embedding: %w", err)
	}

	// Calculate weighted average similarity with interests
	var totalScore float64
	var totalWeight float64

	for _, interest := range interests {
		// Get or generate interest embedding
		var interestEmb []float64
		if len(interest.Embedding) > 0 {
			if err := json.Unmarshal(interest.Embedding, &interestEmb); err != nil {
				return 0, fmt.Errorf("unmarshaling interest embedding: %w", err)
			}
		} else {
			// Generate and cache embedding
			interestEmb, err = c.GetEmbedding(interest.Description)
			if err != nil {
				fmt.Printf("Warning: failed to get embedding for interest '%s': %v\n", interest.Description, err)
				continue
			}

			// Cache embedding
			embData, _ := json.Marshal(interestEmb)
			interest.Embedding = embData
		}

		similarity := CosineSimilarity(articleEmb, interestEmb)
		totalScore += similarity * interest.Weight
		totalWeight += interest.Weight
	}

	if totalWeight == 0 {
		return 0, nil
	}

	return totalScore / totalWeight, nil
}

// ScoreAllUnscored scores all articles that have a relevance score of 0
func (c *Client) ScoreAllUnscored(maxAgeDays int) error {
	interests, err := c.db.GetInterests()
	if err != nil {
		return fmt.Errorf("getting interests: %w", err)
	}

	if len(interests) == 0 {
		fmt.Println("No interests configured, skipping scoring")
		return nil
	}

	// Get unread articles
	articles, err := c.db.GetUnreadArticles(24 * time.Duration(maxAgeDays))
	if err != nil {
		return fmt.Errorf("getting articles: %w", err)
	}

	for i, article := range articles {
		// Skip already scored articles
		if article.RelevanceScore > 0 {
			continue
		}

		score, err := c.ScoreArticle(&article, interests)
		if err != nil {
			fmt.Printf("Warning: failed to score article '%s': %v\n", article.Title, err)
			continue
		}

		if err := c.db.UpdateArticleRelevance(article.ID, score); err != nil {
			fmt.Printf("Warning: failed to update article relevance: %v\n", err)
		}

		fmt.Printf("Scored %d/%d articles\r", i+1, len(articles))
	}
	fmt.Println()

	return nil
}
