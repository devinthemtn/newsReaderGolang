package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/thomaskoefod/newsreadr/pkg/models"
)

type articleItem struct {
	article models.Article
}

func (i articleItem) Title() string {
	return i.article.Title
}

func (i articleItem) Description() string {
	return fmt.Sprintf("%.2f | %s", i.article.RelevanceScore, i.article.PublishedAt.Format("Jan 2, 2006"))
}

func (i articleItem) FilterValue() string {
	return i.article.Title
}

var _ list.Item = articleItem{}
