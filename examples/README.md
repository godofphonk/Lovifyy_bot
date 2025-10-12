# Configuration Examples

This folder contains example configuration files to help you get started with Lovifyy Bot.

## Files:

### `.env.example`
Template for environment variables. Copy to `.env` and fill in your values:
```bash
cp examples/.env.example .env
# Edit .env with your tokens
```

### `config.example.json`
Template for JSON configuration. Copy to `config.json` and customize:
```bash
cp examples/config.example.json config.json
# Edit config.json with your settings
```

## Quick Start:

1. **Copy configuration files:**
   ```bash
   cp examples/.env.example .env
   cp examples/config.example.json config.json
   ```

2. **Edit with your credentials:**
   - Get Telegram Bot Token from [@BotFather](https://t.me/botfather)
   - Get OpenAI API Key from [OpenAI Platform](https://platform.openai.com)
   - Add your admin user IDs

3. **Run the bot:**
   ```bash
   make run
   # or
   go run cmd/main.go
   ```

## Configuration Options:

See the [API Documentation](../docs/API.md) for detailed configuration options.
