package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/thomaskoefod/newsreadr/pkg/models"
)

// AddFeed inserts a new feed
func (db *DB) AddFeed(feed *models.Feed) error {
	result, err := db.Exec(
		"INSERT INTO feeds (url, name, enabled, created_at) VALUES (?, ?, ?, ?)",
		feed.URL, feed.Name, feed.Enabled, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("inserting feed: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}

	feed.ID = id
	return nil
}

// GetFeeds retrieves all feeds
func (db *DB) GetFeeds() ([]models.Feed, error) {
	rows, err := db.Query("SELECT id, url, name, enabled, created_at FROM feeds ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("querying feeds: %w", err)
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var feed models.Feed
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Name, &feed.Enabled, &feed.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

// GetEnabledFeeds retrieves only enabled feeds
func (db *DB) GetEnabledFeeds() ([]models.Feed, error) {
	rows, err := db.Query("SELECT id, url, name, enabled, created_at FROM feeds WHERE enabled = 1 ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("querying enabled feeds: %w", err)
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var feed models.Feed
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Name, &feed.Enabled, &feed.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

// UpdateFeed updates an existing feed
func (db *DB) UpdateFeed(feed *models.Feed) error {
	_, err := db.Exec(
		"UPDATE feeds SET url = ?, name = ?, enabled = ? WHERE id = ?",
		feed.URL, feed.Name, feed.Enabled, feed.ID,
	)
	if err != nil {
		return fmt.Errorf("updating feed: %w", err)
	}
	return nil
}

// DeleteFeed removes a feed and its articles
func (db *DB) DeleteFeed(id int64) error {
	_, err := db.Exec("DELETE FROM feeds WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting feed: %w", err)
	}
	return nil
}

// AddArticle inserts a new article
func (db *DB) AddArticle(article *models.Article) error {
	result, err := db.Exec(
		"INSERT INTO articles (feed_id, title, url, content, description, published_at, fetched_at, relevance_score) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		article.FeedID, article.Title, article.URL, article.Content, article.Description, article.PublishedAt, time.Now(), article.RelevanceScore,
	)
	if err != nil {
		return fmt.Errorf("inserting article: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}

	article.ID = id
	return nil
}

// GetUnreadArticles retrieves articles not marked as read, newer than maxAge, ordered by relevance
func (db *DB) GetUnreadArticles(maxAge time.Duration) ([]models.Article, error) {
	cutoff := time.Now().Add(-maxAge)
	query := `
		SELECT a.id, a.feed_id, a.title, a.url, a.content, a.description, a.published_at, a.fetched_at, a.relevance_score
		FROM articles a
		LEFT JOIN read_articles r ON a.id = r.article_id
		WHERE r.article_id IS NULL AND a.published_at >= ?
		ORDER BY a.relevance_score DESC, a.published_at DESC
	`

	rows, err := db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("querying unread articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		if err := rows.Scan(&article.ID, &article.FeedID, &article.Title, &article.URL, &article.Content, &article.Description, &article.PublishedAt, &article.FetchedAt, &article.RelevanceScore); err != nil {
			return nil, fmt.Errorf("scanning article: %w", err)
		}
		articles = append(articles, article)
	}

	return articles, rows.Err()
}

// GetArticleByID retrieves a single article
func (db *DB) GetArticleByID(id int64) (*models.Article, error) {
	var article models.Article
	err := db.QueryRow(
		"SELECT id, feed_id, title, url, content, description, published_at, fetched_at, relevance_score FROM articles WHERE id = ?",
		id,
	).Scan(&article.ID, &article.FeedID, &article.Title, &article.URL, &article.Content, &article.Description, &article.PublishedAt, &article.FetchedAt, &article.RelevanceScore)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying article: %w", err)
	}

	return &article, nil
}

// MarkArticleRead marks an article as read
func (db *DB) MarkArticleRead(articleID int64) error {
	_, err := db.Exec(
		"INSERT INTO read_articles (article_id, read_at) VALUES (?, ?)",
		articleID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("marking article as read: %w", err)
	}
	return nil
}

// DeleteReadArticles removes read articles from database
func (db *DB) DeleteReadArticles() error {
	_, err := db.Exec("DELETE FROM articles WHERE id IN (SELECT article_id FROM read_articles)")
	if err != nil {
		return fmt.Errorf("deleting read articles: %w", err)
	}
	return nil
}

// DeleteOldArticles removes articles older than maxAge
func (db *DB) DeleteOldArticles(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	_, err := db.Exec("DELETE FROM articles WHERE published_at < ?", cutoff)
	if err != nil {
		return fmt.Errorf("deleting old articles: %w", err)
	}
	return nil
}

// AddInterest inserts a new user interest
func (db *DB) AddInterest(interest *models.UserInterest) error {
	result, err := db.Exec(
		"INSERT INTO user_interests (description, weight, embedding) VALUES (?, ?, ?)",
		interest.Description, interest.Weight, interest.Embedding,
	)
	if err != nil {
		return fmt.Errorf("inserting interest: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}

	interest.ID = id
	return nil
}

// GetInterests retrieves all user interests
func (db *DB) GetInterests() ([]models.UserInterest, error) {
	rows, err := db.Query("SELECT id, description, weight, embedding FROM user_interests")
	if err != nil {
		return nil, fmt.Errorf("querying interests: %w", err)
	}
	defer rows.Close()

	var interests []models.UserInterest
	for rows.Next() {
		var interest models.UserInterest
		var embedding sql.NullString
		if err := rows.Scan(&interest.ID, &interest.Description, &interest.Weight, &embedding); err != nil {
			return nil, fmt.Errorf("scanning interest: %w", err)
		}
		if embedding.Valid {
			interest.Embedding = []byte(embedding.String)
		}
		interests = append(interests, interest)
	}

	return interests, rows.Err()
}

// UpdateArticleRelevance updates the relevance score of an article
func (db *DB) UpdateArticleRelevance(articleID int64, score float64) error {
	_, err := db.Exec("UPDATE articles SET relevance_score = ? WHERE id = ?", score, articleID)
	if err != nil {
		return fmt.Errorf("updating article relevance: %w", err)
	}
	return nil
}
