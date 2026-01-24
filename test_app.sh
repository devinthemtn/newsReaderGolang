#!/bin/bash

# Test script for NewsReadr application
# This script tests basic functionality without requiring Ollama

set -e

echo "=== NewsReadr Application Test ==="
echo ""

# Build the application
echo "1. Building the application..."
go build -o newsreadr cmd/newsreadr/main.go
echo "✓ Build successful"

# Test config creation (if not exists)
CONFIG_PATH="$HOME/.config/newsreader/config.yaml"
if [ ! -f "$CONFIG_PATH" ]; then
    echo ""
    echo "2. Testing config creation..."
    timeout 2s ./newsreadr || true
    if [ -f "$CONFIG_PATH" ]; then
        echo "✓ Config file created successfully at $CONFIG_PATH"
    else
        echo "✗ Config file creation failed"
        exit 1
    fi
else
    echo "✓ Config file already exists at $CONFIG_PATH"
fi

# Check database creation
DB_PATH="$HOME/.config/newsreader/data.db"
if [ -f "$DB_PATH" ]; then
    echo "✓ Database file created at $DB_PATH"

    # Check database tables (requires sqlite3)
    if command -v sqlite3 &> /dev/null; then
        echo ""
        echo "3. Checking database schema..."
        TABLES=$(sqlite3 "$DB_PATH" ".tables")
        echo "Database tables: $TABLES"

        # Check if required tables exist
        if echo "$TABLES" | grep -q "feeds" && echo "$TABLES" | grep -q "articles"; then
            echo "✓ Required database tables exist"
        else
            echo "✗ Missing required database tables"
            exit 1
        fi
    else
        echo "! sqlite3 not available, skipping database schema check"
    fi
else
    echo "✗ Database file not found"
    exit 1
fi

echo ""
echo "4. Testing application launch..."
# Test that the app can start (will exit after 2 seconds)
timeout 2s ./newsreadr &> /tmp/newsreadr_test.log || true

# Check if there were any critical errors
if grep -q "panic\|fatal" /tmp/newsreadr_test.log; then
    echo "✗ Application encountered critical errors:"
    cat /tmp/newsreadr_test.log
    exit 1
else
    echo "✓ Application starts without critical errors"
fi

# Show any warnings (expected for missing Ollama)
if grep -q "Warning" /tmp/newsreadr_test.log; then
    echo ""
    echo "Warnings (expected if Ollama is not installed):"
    grep "Warning" /tmp/newsreadr_test.log || true
fi

echo ""
echo "=== Test Summary ==="
echo "✓ Application builds successfully"
echo "✓ Configuration system works"
echo "✓ Database initialization works"
echo "✓ TUI launches without crashes"
echo ""
echo "Next steps:"
echo "1. Install Ollama for AI-powered article scoring:"
echo "   curl https://ollama.ai/install.sh | sh"
echo "   ollama pull llama2"
echo ""
echo "2. Edit your config file to add more RSS feeds:"
echo "   $CONFIG_PATH"
echo ""
echo "3. Run the application:"
echo "   ./newsreadr"
echo ""
echo "Keyboard shortcuts in the app:"
echo "- 'f' or 'F' to fetch articles from RSS feeds"
echo "- 'r' to refresh the article list"
echo "- Enter to read an article"
echo "- 'o' to open article in browser"
echo "- '?' to show help"
echo "- 'q' or Ctrl+C to quit"

# Cleanup
rm -f /tmp/newsreadr_test.log

echo ""
echo "Test completed successfully! 🎉"
