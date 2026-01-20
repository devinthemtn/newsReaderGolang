# NewsReadr

A Terminal User Interface (TUI) application for reading and filtering news articles based on your interests using AI-powered semantic matching.

## Features

- 📰 **RSS Feed Support**: Fetch articles from multiple RSS feeds
- 🤖 **AI-Powered Filtering**: Uses local LLM (Ollama) for semantic matching based on your interests
- 📅 **Fresh Content**: Only shows articles less than 2 weeks old
- 🗑️ **Auto-Delete**: Automatically deletes articles after reading
- 🌐 **Dual Viewing**: View articles in TUI or open in browser
- 💾 **Raindrop.io Integration**: Save articles to Raindrop.io with one keystroke
- ⌨️ **Keyboard-Driven**: Fully navigable with keyboard shortcuts
- 🎨 **Beautiful TUI**: Built with Charm libraries for a polished terminal experience

## Prerequisites

- Go 1.21 or later
- [Ollama](https://ollama.ai/) running locally (for AI filtering)
- A Raindrop.io account and API token (optional, for saving articles)

## Installation

```bash
# Clone the repository
git clone https://github.com/thomaskoefod/newsreadr.git
cd newsreadr

# Build the application
go build -o newsreadr cmd/newsreadr/main.go

# Move to PATH (optional)
sudo mv newsreadr /usr/local/bin/
```

## Quick Start

1. **Install Ollama** (if not already installed):
   ```bash
   curl https://ollama.ai/install.sh | sh
   ollama pull llama2
   ```

2. **Run NewsReadr** for the first time:
   ```bash
   ./newsreadr
   ```
   
   This will create a default configuration file at `~/.config/newsreader/config.yaml`.

3. **Configure your interests** by editing `~/.config/newsreader/config.yaml`:
   ```yaml
   interests:
     - "artificial intelligence and machine learning"
     - "golang programming and software development"
     - "climate change technology"
   ```

4. **Fetch articles**:
   Press `f` to fetch articles from your configured feeds. The app will:
   - Download articles from RSS feeds
   - Filter out articles older than 2 weeks
   - Score each article based on your interests using AI
   - Display them ordered by relevance

## Configuration

The configuration file is located at `~/.config/newsreader/config.yaml`. See `config.example.yaml` for a complete example.

### Adding Feeds

```yaml
feeds:
  - url: https://hnrss.org/frontpage
    name: Hacker News
  - url: https://www.theverge.com/rss/index.xml
    name: The Verge
```

### Setting Your Interests

```yaml
interests:
  - "machine learning and AI research"
  - "web development with React and TypeScript"
  - "sustainable energy solutions"
```

### Raindrop.io Integration

To enable Raindrop.io integration:

1. Get your API token from [Raindrop.io Settings](https://app.raindrop.io/settings/integrations)
2. Add it to your config:
   ```yaml
   raindrop:
     api_token: your_token_here
   ```

## Keyboard Shortcuts

### Article List View
- `↑/↓` or `j/k` - Navigate articles
- `Enter` - Read article
- `o` - Open article in browser
- `r` - Refresh article list
- `f` - Fetch new articles from feeds
- `/` - Filter articles
- `?` - Show help
- `q` or `Ctrl+C` - Quit

### Article Detail View
- `Enter` - Mark as read and delete article
- `o` - Open article in browser
- `s` - Save article to Raindrop.io
- `Esc` - Back to list
- `?` - Show help
- `q` or `Ctrl+C` - Quit

## How It Works

1. **Fetching**: NewsReadr fetches articles from your configured RSS feeds
2. **Filtering**: Articles older than the configured age (default: 14 days) are filtered out
3. **AI Scoring**: Each article is scored against your interests using semantic similarity via Ollama embeddings
4. **Display**: Articles are displayed ordered by relevance score
5. **Reading**: When you read an article (press Enter), it's marked as read and automatically deleted
6. **Cleanup**: Old articles are periodically cleaned up from the database

## Architecture

```
newsreadr/
├── cmd/newsreadr/          # Main application entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── database/           # SQLite operations
│   ├── feed/               # RSS feed fetching & parsing
│   ├── ai/                 # Ollama integration & filtering
│   ├── raindrop/           # Raindrop.io API client
│   └── tui/                # Bubble Tea UI components
└── pkg/models/             # Shared data models
```

## Troubleshooting

### Ollama Connection Error
Make sure Ollama is running:
```bash
ollama serve
```

### No Articles Showing
1. Press `f` to fetch articles
2. Check that your feeds are valid RSS feeds
3. Verify articles are less than 2 weeks old

### Low Relevance Scores
The AI scoring is based on semantic similarity to your interests. Try:
- Making your interests more specific
- Adding more varied interest descriptions
- Using a different Ollama model (e.g., `mistral`)

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
