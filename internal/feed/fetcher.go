package feed

import (
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/thomaskoefod/newsreadr/internal/database"
	"github.com/thomaskoefod/newsreadr/pkg/models"
)

type Fetcher struct {
	db     *database.DB
	parser *gofeed.Parser
}

func NewFetcher(db *database.DB) *Fetcher {
	return &Fetcher{
		db:     db,
		parser: gofeed.NewParser(),
	}
}

// FetchFeed fetches and parses an RSS feed
func (f *Fetcher) FetchFeed(feedURL string) (*gofeed.Feed, error) {
	feed, err := f.parser.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("parsing feed %s: %w", feedURL, err)
	}
	return feed, nil
}

// FetchAndStore fetches a feed and stores new articles in the database
func (f *Fetcher) FetchAndStore(feed *models.Feed) (int, error) {
	rssFeed, err := f.FetchFeed(feed.URL)
	if err != nil {
		return 0, err
	}

	newArticles := 0
	for _, item := range rssFeed.Items {
		article := f.convertToArticle(item, feed.ID)
		if article == nil {
			continue
		}

		// Try to insert, ignore duplicates (unique URL constraint)
		if err := f.db.AddArticle(article); err != nil {
			// Skip if duplicate
			continue
		}
		newArticles++
	}

	return newArticles, nil
}

// FetchAllFeeds fetches all enabled feeds
func (f *Fetcher) FetchAllFeeds() (int, error) {
	feeds, err := f.db.GetEnabledFeeds()
	if err != nil {
		return 0, fmt.Errorf("getting enabled feeds: %w", err)
	}

	totalNew := 0
	for _, feed := range feeds {
		count, err := f.FetchAndStore(&feed)
		if err != nil {
			// Log error but continue with other feeds
			fmt.Printf("Error fetching feed %s: %v\n", feed.Name, err)
			continue
		}
		totalNew += count
	}

	return totalNew, nil
}

// convertToArticle converts a gofeed.Item to our Article model
func (f *Fetcher) convertToArticle(item *gofeed.Item, feedID int64) *models.Article {
	// Determine published date
	var publishedAt time.Time
	if item.PublishedParsed != nil {
		publishedAt = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		publishedAt = *item.UpdatedParsed
	} else {
		// Skip articles without dates
		return nil
	}

	// Get content (prefer content over description)
	content := ""
	if item.Content != "" {
		content = item.Content
	} else if item.Description != "" {
		content = item.Description
	}

	description := item.Description
	if description == "" && item.Content != "" {
		// Truncate content for description if needed
		if len(item.Content) > 500 {
			description = item.Content[:500] + "..."
		} else {
			description = item.Content
		}
	}

	return &models.Article{
		FeedID:      feedID,
		Title:       item.Title,
		URL:         item.Link,
		Content:     content,
		Description: description,
		PublishedAt: publishedAt,
	}
}
