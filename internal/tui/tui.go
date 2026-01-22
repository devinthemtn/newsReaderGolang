package tui

import (
	"fmt"
	"strings"
	"time"

	html2md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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
	allArticles []models.Article // Keep unfiltered list
	list       list.Model
	viewport   viewport.Model
	filterInput textinput.Model
	isFiltering bool
	cursor     int
	width      int
	height     int
	err        error
	statusMsg  string
	articleContent string
	renderer   *glamour.TermRenderer
	mdConverter *html2md.Converter
	ready      bool
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

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
)

func New(cfg *config.Config, db *database.DB, fetcher *feed.Fetcher, aiClient *ai.Client, rdClient *raindrop.Client) Model {
	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "NewsReadr - Your Personalized News"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false) // Disable built-in filtering, we'll use our own
	l.Styles.Title = titleStyle

	// Create glamour renderer for markdown
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	// Create HTML to Markdown converter
	converter := html2md.NewConverter("", true, nil)

	// Create filter input
	ti := textinput.New()
	ti.Placeholder = "Type to filter articles..."
	ti.CharLimit = 100
	ti.Width = 50

	return Model{
		cfg:         cfg,
		db:          db,
		fetcher:     fetcher,
		aiClient:    aiClient,
		rdClient:    rdClient,
		view:        ViewArticleList,
		list:        l,
		renderer:    renderer,
		mdConverter: converter,
		filterInput: ti,
		isFiltering: false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadArticles(m.db, m.cfg),
		fetchFeeds(m.fetcher, m.db, m.aiClient, m.cfg),
		tea.EnterAltScreen,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}
		
		return m, nil

	case tea.KeyMsg:
		// Handle filter input first if we're in filtering mode
		if m.isFiltering && m.view == ViewArticleList {
			switch msg.String() {
			case "esc":
				m.isFiltering = false
				m.filterInput.SetValue("")
				m.filterInput.Blur()
				// Reset to all articles
				m.articles = m.allArticles
				items := make([]list.Item, len(m.articles))
				for i, article := range m.articles {
					items[i] = articleItem{article}
				}
				m.list.SetItems(items)
				m.statusMsg = fmt.Sprintf("Showing all %d articles", len(m.articles))
				return m, nil
			case "enter":
				m.isFiltering = false
				m.filterInput.Blur()
				m.statusMsg = fmt.Sprintf("Filtered to %d articles", len(m.articles))
				return m, nil
			default:
				// Pass input to the textinput
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.applyFilter()
				return m, cmd
			}
		}
		
		return m.handleKeyPress(msg)

	case articlesLoadedMsg:
		m.articles = msg.articles
		m.allArticles = msg.articles // Store unfiltered list
		items := make([]list.Item, len(m.articles))
		for i, article := range m.articles {
			items[i] = articleItem{article}
		}
		m.list.SetItems(items)
		m.list.ResetSelected()
		m.statusMsg = fmt.Sprintf("Loaded %d articles", len(m.articles))
		return m, nil

	case errorMsg:
		m.err = msg.err
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		return m, nil
	}

	if m.view == ViewArticleDetail {
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	
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
			content := m.formatArticleForView(i.article)
			m.articleContent = content
			m.viewport.SetContent(content)
			m.viewport.GotoTop()
			return m, nil
		}

	case "/", "f":
		m.isFiltering = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case "r":
		return m, tea.Batch(
			loadArticles(m.db, m.cfg),
			func() tea.Msg { return statusMsg("Refreshing articles...") },
		)

	case "F":
		return m, tea.Batch(
			fetchFeeds(m.fetcher, m.db, m.aiClient, m.cfg),
			func() tea.Msg { return statusMsg("Fetching new articles...") },
		)

	case "d":
		return m, tea.Batch(
			deleteOldArticles(m.db, m.cfg),
			func() tea.Msg { return statusMsg("Deleting old articles...") },
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
	
	// Scroll controls
	case "up", "k":
		m.viewport.LineUp(1)
		return m, nil
	case "down", "j":
		m.viewport.LineDown(1)
		return m, nil
	case "pgup", "b":
		m.viewport.ViewUp()
		return m, nil
	case "pgdown", "f", " ":
		m.viewport.ViewDown()
		return m, nil
	case "home", "g":
		m.viewport.GotoTop()
		return m, nil
	case "end", "G":
		m.viewport.GotoBottom()
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

	// Show filter input if active
	if m.isFiltering {
		s.WriteString(filterStyle.Render("Filter: "))
		s.WriteString(m.filterInput.View())
		s.WriteString(helpStyle.Render(" (enter: apply, esc: cancel)"))
		s.WriteString("\n\n")
	}

	s.WriteString(m.list.View())
	s.WriteString("\n")

	// Status bar
	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.statusMsg != "" {
		s.WriteString(statusStyle.Render(m.statusMsg))
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("enter: read • o: open browser • /,f: filter • r: refresh • F: fetch new • d: delete old • ?: help • q: quit"))

	return s.String()
}

func (m Model) renderDetail() string {
	var s strings.Builder

	s.WriteString(m.viewport.View())
	s.WriteString("\n")

	// Scroll indicator
	scrollInfo := helpStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	s.WriteString(scrollInfo)
	s.WriteString(" ")

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n")
	} else if m.statusMsg != "" {
		s.WriteString(statusStyle.Render(m.statusMsg))
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("↑/↓,j/k: scroll • pgup/pgdn,space: page • enter: mark read • o: browser • s: raindrop • esc: back"))

	return s.String()
}

func (m Model) renderHelp() string {
	help := `
NewsReadr - Keyboard Shortcuts

Article List:
  ↑/↓, j/k     Navigate articles
  enter        Read article
  o            Open article in browser
  /,f          Quick filter by title
  r            Refresh article list
  F            Fetch new articles from feeds
  d            Delete old articles (older than configured max age)
  q, ctrl+c    Quit

Filter Mode:
  type         Filter articles by title
  enter        Apply filter and exit filter mode
  esc          Cancel filter and show all articles

Article Detail:
  ↑/↓, j/k     Scroll line by line
  pgup/pgdn    Scroll page by page
  space        Page down
  home/g       Go to top
  end/G        Go to bottom
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

func deleteOldArticles(db *database.DB, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		maxAge := time.Duration(cfg.UI.ArticleMaxAgeDays) * 24 * time.Hour
		
		// Get count before deletion for reporting
		articles, _ := db.GetUnreadArticles(maxAge * 10) // Get articles older than max age
		oldCount := 0
		cutoff := time.Now().Add(-maxAge)
		for _, article := range articles {
			if article.PublishedAt.Before(cutoff) {
				oldCount++
			}
		}
		
		// Delete old articles
		if err := db.DeleteOldArticles(maxAge); err != nil {
			return errorMsg{err}
		}
		
		// Also delete read articles
		if err := db.DeleteReadArticles(); err != nil {
			return errorMsg{err}
		}
		
		// Reload articles after deletion
		articles, err := db.GetUnreadArticles(maxAge)
		if err != nil {
			return errorMsg{err}
		}
		
		return articlesLoadedMsg{articles}
	}
}

func (m Model) formatArticleForView(article models.Article) string {
	var s strings.Builder

	// Convert HTML content to Markdown
	content := article.Content
	if content != "" {
		// Try to convert HTML to markdown
		markdown, err := m.mdConverter.ConvertString(content)
		if err == nil {
			content = markdown
		}
	}

	// If no content, use description
	if content == "" {
		content = article.Description
		if content != "" {
			markdown, err := m.mdConverter.ConvertString(content)
			if err == nil {
				content = markdown
			}
		}
	}

	// Render the markdown with glamour
	rendered, err := m.renderer.Render(content)
	if err != nil {
		// Fallback to plain text if rendering fails
		s.WriteString(articleTitleStyle.Render(article.Title))
		s.WriteString("\n")
		s.WriteString(helpStyle.Render(fmt.Sprintf("Published: %s | Score: %.2f", article.PublishedAt.Format("Jan 2, 2006"), article.RelevanceScore)))
		s.WriteString("\n\n")
		s.WriteString(content)
		return s.String()
	}

	// Build the article view with rendered content
	s.WriteString(articleTitleStyle.Render(article.Title))
	s.WriteString("\n")
	s.WriteString(helpStyle.Render(fmt.Sprintf("Published: %s | Score: %.2f | URL: %s", 
		article.PublishedAt.Format("Jan 2, 2006"), 
		article.RelevanceScore,
		article.URL)))
	s.WriteString("\n\n")
	s.WriteString(rendered)

	return s.String()
}

// applyFilter filters articles based on the filter input
func (m *Model) applyFilter() {
	filterText := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	
	if filterText == "" {
		// No filter, show all articles
		m.articles = m.allArticles
	} else {
		// Filter articles by title
		filtered := []models.Article{}
		for _, article := range m.allArticles {
			if strings.Contains(strings.ToLower(article.Title), filterText) {
				filtered = append(filtered, article)
			}
		}
		m.articles = filtered
	}
	
	// Update list items
	items := make([]list.Item, len(m.articles))
	for i, article := range m.articles {
		items[i] = articleItem{article}
	}
	m.list.SetItems(items)
	m.list.ResetSelected()
}
