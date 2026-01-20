package models

import "time"

type Feed struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type Article struct {
	ID             int64     `json:"id"`
	FeedID         int64     `json:"feed_id"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	Content        string    `json:"content"`
	Description    string    `json:"description"`
	PublishedAt    time.Time `json:"published_at"`
	FetchedAt      time.Time `json:"fetched_at"`
	RelevanceScore float64   `json:"relevance_score"`
}

type UserInterest struct {
	ID          int64   `json:"id"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Embedding   []byte  `json:"embedding,omitempty"`
}

type ReadArticle struct {
	ArticleID int64     `json:"article_id"`
	ReadAt    time.Time `json:"read_at"`
}
