package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thomaskoefod/newsreadr/internal/ai"
	"github.com/thomaskoefod/newsreadr/internal/config"
	"github.com/thomaskoefod/newsreadr/internal/database"
	"github.com/thomaskoefod/newsreadr/internal/feed"
	"github.com/thomaskoefod/newsreadr/internal/raindrop"
	"github.com/thomaskoefod/newsreadr/pkg/models"
)

type View int

const (
	ViewArticleList View = iota
	ViewArticleDetail
	ViewHelp
)

type Model struct {
	cfg        *config.Config
	db         *database.DB
	fetcher    *feed.Fetcher
	aiClient   *ai.Client
	rdClient   *raindrop.Client
	view       View
	articles   []models.Article
	list       list.Model
	cursor     int
	width      int
	height     int
	err        error
	statusMsg  string
	articleContent string
}

type articlesLoadedMsg struct {
	articles []models.Article
}

type errorMsg struct {
	err error
}

type statusMsg string

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	articleTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("86")).
				MarginBottom(1)
)

func New(cfg *config.Config, db *database.DB, fetcher *feed.Fetcher, aiClient *ai.Client, rdClient *raindrop.Client) Model {
	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "NewsReadr - Your Personalized News"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	return Model{
		cfg:      cfg,
		db:       db,
		fetcher:  fetcher,
		aiClient: aiClient,
		rdClient: rdClient,
		view:     ViewArticleList,
		list:     l,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadArticles(m.db, m.cfg),
		tea.EnterAltScreen,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case articlesLoadedMsg:
		m.articles = msg.articles
		items := make([]list.Item, len(m.articles))
		for i, article := range m.articles {
			items[i] = articleItem{article}
		}
		m.list.SetItems(items)
		m.statusMsg = fmt.Sprintf("Loaded %d articles", len(m.articles))
		return m, nil

	case errorMsg:
		m.err = msg.err
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewArticleList:
		return m.handleListKeys(msg)
	case ViewArticleDetail:
		return m.handleDetailKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	}
	return m, nil
}

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "enter":
		if i, ok := m.list.SelectedItem().(articleItem); ok {
			m.view = ViewArticleDetail
			m.articleContent = formatArticleForView(i.article)
			return m, nil
		}

	case "r":
		return m, tea.Batch(
			loadArticles(m.db, m.cfg),
			func() tea.Msg { return statusMsg("Refreshing articles...") },
		)

	case "f":
		return m, tea.Batch(
			fetchFeeds(m.fetcher, m.db, m.aiClient, m.cfg),
			func() tea.Msg { return statusMsg("Fetching new articles...") },
		)

	case "?":
		m.view = ViewHelp
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc", "backspace":
		m.view = ViewArticleList
		return m, nil

	case "enter":
		// Mark as read and delete
		if i, ok := m.list.SelectedItem().(articleItem); ok {
			m.db.MarkArticleRead(i.article.ID)
			m.db.DeleteReadArticles()
			m.view = ViewArticleList
			return m, tea.Batch(
				loadArticles(m.db, m.cfg),
				func() tea.Msg { return statusMsg("Article marked as read") },
			)
		}

	case "o":
		// Open in browser
		if i, ok := m.list.SelectedItem().(articleItem); ok {
			openBrowser(i.article.URL)
			return m, func() tea.Msg { return statusMsg("Opened in browser") }
		}

	case "s":
		// Send to Raindrop
		if i, ok := m.list.SelectedItem().(articleItem); ok {
			if err := m.rdClient.SaveArticle(&i.article); err != nil {
				return m, func() tea.Msg { return errorMsg{err} }
			}
			return m, func() tea.Msg { return statusMsg("Saved to Raindrop.io") }
		}

	case "?":
		m.view = ViewHelp
		return m, nil
	}

	return m, nil
}

func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "q":
		if m.articleContent != "" {
			m.view = ViewArticleDetail
		} else {
			m.view = ViewArticleList
		}
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case ViewArticleList:
		return m.renderList()
	case ViewArticleDetail:
		return m.renderDetail()
	case ViewHelp:
		return m.renderHelp()
	}
	return ""
}

func (m Model) renderList() string {
	var s strings.Builder

	s.WriteString(m.list.View())
	s.WriteString("\n")

	// Status bar
	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.statusMsg != "" {
		s.WriteString(statusStyle.Render(m.statusMsg))
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("enter: read • o: open browser • r: refresh • f: fetch new • ?: help • q: quit"))

	return s.String()
}

func (m Model) renderDetail() string {
	var s strings.Builder

	s.WriteString(m.articleContent)
	s.WriteString("\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n")
	} else if m.statusMsg != "" {
		s.WriteString(statusStyle.Render(m.statusMsg))
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("enter: mark read & delete • o: open browser • s: save to Raindrop • esc: back • ?: help • q: quit"))

	return s.String()
}

func (m Model) renderHelp() string {
	help := `
NewsReadr - Keyboard Shortcuts

Article List:
  ↑/↓, j/k     Navigate articles
  enter        Read article
  o            Open article in browser
  r            Refresh article list
  f            Fetch new articles from feeds
  /            Filter articles
  q, ctrl+c    Quit

Article Detail:
  enter        Mark as read and delete article
  o            Open article in browser
  s            Save article to Raindrop.io
  esc          Back to list
  q, ctrl+c    Quit

General:
  ?            Show/hide this help
`
	return help + "\n" + helpStyle.Render("Press ? or esc to close help")
}

func loadArticles(db *database.DB, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		maxAge := time.Duration(cfg.UI.ArticleMaxAgeDays) * 24 * time.Hour
		articles, err := db.GetUnreadArticles(maxAge)
		if err != nil {
			return errorMsg{err}
		}
		return articlesLoadedMsg{articles}
	}
}

func fetchFeeds(fetcher *feed.Fetcher, db *database.DB, aiClient *ai.Client, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		count, err := fetcher.FetchAllFeeds()
		if err != nil {
			return errorMsg{err}
		}

		// Score new articles
		if err := aiClient.ScoreAllUnscored(cfg.UI.ArticleMaxAgeDays); err != nil {
			return errorMsg{err}
		}

		// Clean up old articles
		maxAge := time.Duration(cfg.UI.ArticleMaxAgeDays) * 24 * time.Hour
		if err := db.DeleteOldArticles(maxAge); err != nil {
			return errorMsg{err}
		}

		return statusMsg(fmt.Sprintf("Fetched %d new articles", count))
	}
}

func formatArticleForView(article models.Article) string {
	var s strings.Builder

	s.WriteString(articleTitleStyle.Render(article.Title))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("Published: %s | Score: %.2f", article.PublishedAt.Format("Jan 2, 2006"), article.RelevanceScore)))
	s.WriteString("\n\n")
	s.WriteString(article.Content)

	return s.String()
}
