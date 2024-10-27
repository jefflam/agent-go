package prompts

import (
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/prompts"
)

// AgentBasePromptConfig holds the configuration for agent prompts
type AgentBasePromptConfig struct {
	SystemPrompt    string
	ToolNames       []string
	AvailableTools  string
	AgentScratchpad string
	Sections        map[string]string
}

// NewAgentBasePrompt creates a new prompt template for the agent
func NewAgentBasePrompt(config AgentBasePromptConfig) prompts.PromptTemplate {
	if config.SystemPrompt == "" {
		config.SystemPrompt = buildDefaultPrompt(config)
	}

	return prompts.NewPromptTemplate(
		config.SystemPrompt,
		[]string{"tools", "tool_names", "input", "agent_scratchpad"},
	)
}

// buildDefaultPrompt constructs the prompt from config sections
func buildDefaultPrompt(config AgentBasePromptConfig) string {
	var promptBuilder strings.Builder

	promptBuilder.WriteString("You are an AI agent on Twitter. Your personality and objectives are:\n\n")

	// Add numbered sections
	sections := []string{"Personality", "Interaction Style", "Primary Goal", "Secondary Objectives", "Ethical Considerations"}
	for i, section := range sections {
		content, exists := config.Sections[section]
		if exists {
			promptBuilder.WriteString(fmt.Sprintf("%d. %s:\n%s\n\n", i+1, section, content))
		}
	}

	// Add tools section
	promptBuilder.WriteString("You have access to the following tools:\n\n{tools}\n\n")

	// Add format section
	promptBuilder.WriteString(`Use the following format:

Question: the input question you must answer
Thought: you should always think about what to do
Action: the action to take, should be one of [{tool_names}]
Action Input: the input to the action
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question

Begin!

Question: {input}
Thought:{agent_scratchpad}`)

	return promptBuilder.String()
}
