package domain

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`

	// Optional: used for tool calling and advanced providers
	Name string `json:"name,omitempty"`
}

type GenerateOptions struct {
	Model       string
	MaxTokens   int
	Temperature float32
	TopP        float32
}

type ChatChunk struct {
	Content string
	Error   error
	Done    bool
}

type ProviderConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type Config struct {
	Providers map[string]ProviderConfig
	Default   string
}
