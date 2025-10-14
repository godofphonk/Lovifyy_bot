# 🤖 Lovifyy Bot

**Enterprise-grade Telegram bot for relationship counseling and psychological research**

A Telegram bot built with Go, featuring AI-powered counseling, structured exercises, and comprehensive admin tools. 
Originally developed for diploma research on attachment styles in relationships.

## 🎯 Key Features

- **🧠 AI Counseling** - Intelligent relationship advice using OpenAI GPT-4o-mini
- **📝 Structured Diary** - Weekly relationship tracking with gender-specific insights  
- **💑 Psychological Exercises** - 4-week program with tips, insights, and joint activities
- **👑 Admin Dashboard** - Complete content management and user analytics
- **📱 Multi-Command Interface** - Both menu commands and inline buttons
- **🔔 Smart Notifications** - Automated reminders with custom scheduling

## 🏗️ Architecture

The project uses modular architecture with 55+ files organized into specialized packages. 
Each module has a single responsibility, with clear separation between bot handlers,
business logic, configuration, and data persistence.

### 📁 Project Structure
```
internal/
├── bot/           # Telegram bot handlers (9 modules)
├── handlers/      # Business logic handlers (5 packages)  
├── config/        # Configuration management (5 modules)
├── history/       # Data persistence layer (5 modules)
├── validator/     # Input validation & sanitization (5 modules)
├── ai/           # OpenAI integration
├── models/       # Data models and user management
└── services/     # Background services
```

## 🛠️ Tech Stack

- **Language:** Go 1.23+
- **AI:** OpenAI GPT-4o-mini API
- **Platform:** Telegram Bot API
- **Deployment:** Docker + Docker Compose
- **Storage:** JSON-based file system
- **Monitoring:** Built-in metrics and health checks
- **Networking:** VPN support for restricted regions

## 📊 Production Features

- **Health checks** on port 8080
- **Metrics endpoint** on port 8081  
- **Structured JSON logging**
- **Graceful shutdown handling**
- **Auto-restart policies**
- **VPN integration** for geo-restricted deployments