# Masa Twitter Agent Framework

A sophisticated Go-based framework for building AI-powered Twitter agents with cognitive processing capabilities, emotional awareness, and natural conversation abilities.

## 🌟 Features

- **Advanced AI Integration**: Built with LangChain GO for powerful language model capabilities
- **Cognitive Processing**: Sophisticated thought generation and contextual awareness
- **Memory Management**: Short and long-term memory systems for contextual conversations
- **Emotional Intelligence**: Built-in emotional state handling for more human-like interactions
- **Twitter API Integration**: Seamless integration with Twitter's API
- **Production Ready**: Built with enterprise-grade Go practices and patterns

## 🚀 Getting Started

### Prerequisites

- Go 1.22 or higher
- Masa Protocol Node or API Access 
- Twitter Developer Account with API credentials
- OpenAI API key

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/agent-go.git
cd agent-go
```

2. Create a `.env` file in the root directory with your credentials:
```env
TWITTER_API_KEY=your_api_key
TWITTER_API_SECRET=your_api_secret
TWITTER_ACCESS_TOKEN=your_access_token
TWITTER_ACCESS_SECRET=your_access_secret
OPENAI_API_KEY=your_openai_key
LOG_LEVEL=INFO
```

3. Install dependencies:
```bash
go mod tidy
```

### Running the Agent

Using make commands:
```bash
# Build and run
make run

# Development mode
make dev

# Build only
make build
```

## 🧠 Core Components

### Thought Processing
The framework includes sophisticated thought processing capabilities:

- **Original Thoughts**: Generates contextually aware original content
- **Mention Replies**: Creates engaging responses to mentions
- **Memory Integration**: Maintains conversation context
- **Personality Traits**: Configurable personality characteristics

### Twitter Integration
Seamless integration with Twitter's API for:

- Mention monitoring
- Tweet publishing
- Conversation threading
- Rate limiting compliance

## 🧪 Testing

Install Ginkgo:
```bash
go install github.com/onsi/ginkgo/v2/ginkgo
```

Run tests:
```bash
# Run all tests
ginkgo -r

# Run integration tests only
ginkgo -r tests/integration

# Watch mode for development
ginkgo watch -r
```

For integration tests, ensure `INTEGRATION_TESTS=true` is set in your `.env` file.

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📚 Documentation

For detailed documentation on components and usage, see the [Wiki](link-to-wiki).

## ⚠️ Note

This is a sophisticated framework designed for production use. Ensure you comply with Twitter's Terms of Service and API usage guidelines when deploying your agent.
