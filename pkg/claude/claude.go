package claude

import (
	"context"
	"fmt"
	"strings"

	claudecode "github.com/yukifoo/claude-code-sdk-go"
)

// TaskExecutionResult represents the result of executing a task with Claude
type TaskExecutionResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	SessionID    string   `json:"sessionId,omitempty"`
	FilesChanged []string `json:"filesChanged,omitempty"`
}

// ClaudeClient wraps the Claude Code SDK for task execution
type ClaudeClient struct {
	workingDirectory string
}

// NewClaudeClient creates a new Claude client with the specified working directory
func NewClaudeClient(workingDirectory string) *ClaudeClient {
	return &ClaudeClient{
		workingDirectory: workingDirectory,
	}
}

// ContinueConversation continues an existing conversation using sessionId
func (c *ClaudeClient) ContinueConversation(sessionId, userMessage string) TaskExecutionResult {
	ctx := context.Background()

	// Create the request to continue the conversation
	request := claudecode.QueryRequest{
		Prompt: userMessage,
		Options: &claudecode.Options{
			MaxTurns:       intPtr(10),
			AllowedTools:   []string{"Read", "Write", "LS", "Grep", "Edit"},
			SystemPrompt:   stringPtr("You are a senior software engineer helping to implement and modify development tasks. Focus on understanding the user's request and making the appropriate changes while maintaining code quality."),
			Cwd:            &c.workingDirectory,
			Verbose:        boolPtr(true),
			PermissionMode: stringPtr("acceptEdits"),
			Resume:         stringPtr(sessionId), // Resume existing session using correct field
		},
	}

	// Execute the request
	messages, err := claudecode.QueryWithRequest(ctx, request)
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to continue conversation with Claude: %v", err),
		}
	}

	if len(messages) == 0 {
		return TaskExecutionResult{
			Success: false,
			Message: "No response received from Claude",
		}
	}

	// Extract response and track file changes
	var filesChanged []string
	responseContent := []string{}

	for _, message := range messages {
		switch msg := message.(type) {
		case *claudecode.ResultMessage:
			for _, block := range msg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					responseContent = append(responseContent, textBlock.Text)
				}
			}
		case *claudecode.AssistantMessage:
			for _, block := range msg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					responseContent = append(responseContent, textBlock.Text)
				}
				if toolBlock, ok := block.(*claudecode.ToolUseBlock); ok {
					// Track file operations
					if toolBlock.Name == "Write" || toolBlock.Name == "Edit" {
						if path, exists := toolBlock.Input["path"]; exists {
							if pathStr, ok := path.(string); ok {
								filesChanged = append(filesChanged, pathStr)
							}
						}
					}
				}
			}
		}
	}

	response := strings.Join(responseContent, "\n")

	return TaskExecutionResult{
		Success:      true,
		Message:      response,
		SessionID:    sessionId, // Return the same session ID
		FilesChanged: removeDuplicates(filesChanged),
	}
}

// extractSessionIDFromResult attempts to extract session ID from a result message
func (c *ClaudeClient) extractSessionIDFromResult(result *claudecode.ResultMessage) string {
	// The session ID is available directly from the ResultMessage
	if result != nil {
		return result.SessionID
	}
	return ""
}

// ExecuteTask runs a task using Claude Code CLI
func (c *ClaudeClient) ExecuteTask(taskID int, taskTitle, taskDescription string) TaskExecutionResult {
	ctx := context.Background()

	// Construct a detailed prompt for Claude
	prompt := fmt.Sprintf(`I need help implementing this specific task:

**Task ID**: %d
**Title**: %s
**Description**: %s

Please analyze the current codebase and implement this task. Consider:
1. The existing code structure and patterns
2. Best practices for the technology stack being used
3. Any dependencies or integration points
4. Testing requirements if applicable

Please implement the necessary code changes to complete this task.`,
		taskID, taskTitle, taskDescription)

	// Create the request using the TypeScript/Python compatible API
	request := claudecode.QueryRequest{
		Prompt: prompt,
		Options: &claudecode.Options{
			MaxTurns:       intPtr(10),
			AllowedTools:   []string{"Read", "Write", "LS", "Grep"},
			SystemPrompt:   stringPtr("You are a senior software engineer helping to implement specific development tasks. Focus on writing high-quality, maintainable code that follows the existing codebase patterns."),
			Cwd:            &c.workingDirectory,
			Verbose:        boolPtr(true),
			PermissionMode: stringPtr("acceptEdits"),
		},
	}

	// Execute the request
	messages, err := claudecode.QueryWithRequest(ctx, request)
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to execute task with Claude: %v", err),
		}
	}

	if len(messages) == 0 {
		return TaskExecutionResult{
			Success: false,
			Message: "No response received from Claude",
		}
	}

	// Extract session ID and analyze the response
	var sessionID string
	var filesChanged []string
	responseContent := []string{}

	for _, message := range messages {
		switch msg := message.(type) {
		case *claudecode.ResultMessage:
			// Try to extract session ID from result if available
			sessionID = c.extractSessionIDFromResult(msg)
			for _, block := range msg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					responseContent = append(responseContent, textBlock.Text)
				}
			}
		case *claudecode.AssistantMessage:
			for _, block := range msg.Content() {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					responseContent = append(responseContent, textBlock.Text)
				}
				if toolBlock, ok := block.(*claudecode.ToolUseBlock); ok {
					// Track file operations
					if toolBlock.Name == "Write" || toolBlock.Name == "Edit" {
						if path, exists := toolBlock.Input["path"]; exists {
							if pathStr, ok := path.(string); ok {
								filesChanged = append(filesChanged, pathStr)
							}
						}
					}
				}
			}
		}
	}

	// Join all response content (for potential future use)
	_ = strings.Join(responseContent, "\n")

	return TaskExecutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Successfully executed task %d. Claude processed %d messages.", taskID, len(messages)),
		SessionID:    sessionID,
		FilesChanged: removeDuplicates(filesChanged),
	}
}

// ExecuteTaskWithStreaming runs a task using Claude Code CLI with streaming
func (c *ClaudeClient) ExecuteTaskWithStreaming(taskID int, taskTitle, taskDescription string) (chan TaskExecutionResult, chan error) {
	resultChan := make(chan TaskExecutionResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		ctx := context.Background()

		// Construct a detailed prompt for Claude
		prompt := fmt.Sprintf(`I need help implementing this specific task:

**Task ID**: %d
**Title**: %s
**Description**: %s

Please analyze the current codebase and implement this task step by step. Consider:
1. The existing code structure and patterns
2. Best practices for the technology stack being used
3. Any dependencies or integration points
4. Testing requirements if applicable

Please implement the necessary code changes to complete this task.`,
			taskID, taskTitle, taskDescription)

		// Create the streaming request
		request := claudecode.QueryRequest{
			Prompt: prompt,
			Options: &claudecode.Options{
				MaxTurns:     intPtr(5),
				AllowedTools: []string{"Read", "Write", "LS", "Grep", "Edit"},
				SystemPrompt: stringPtr("You are a senior software engineer helping to implement specific development tasks. Focus on writing high-quality, maintainable code that follows the existing codebase patterns."),
				Cwd:          &c.workingDirectory,
				OutputFormat: outputFormatPtr(claudecode.OutputFormatStreamJSON),
				Verbose:      boolPtr(true),
			},
		}

		// Execute the streaming request
		messageChan, errChan := claudecode.QueryStreamWithRequest(ctx, request)

		var filesChanged []string
		messageCount := 0

		for {
			select {
			case message, ok := <-messageChan:
				if !ok {
					// Streaming completed
					resultChan <- TaskExecutionResult{
						Success:      true,
						Message:      fmt.Sprintf("Successfully executed task %d with streaming. Processed %d messages.", taskID, messageCount),
						FilesChanged: removeDuplicates(filesChanged),
					}
					return
				}

				messageCount++

				// Track file operations from tool use blocks
				if assistantMsg, ok := message.(*claudecode.AssistantMessage); ok {
					for _, block := range assistantMsg.Content() {
						if toolBlock, ok := block.(*claudecode.ToolUseBlock); ok {
							if toolBlock.Name == "Write" || toolBlock.Name == "Edit" {
								if path, exists := toolBlock.Input["path"]; exists {
									if pathStr, ok := path.(string); ok {
										filesChanged = append(filesChanged, pathStr)
									}
								}
							}
						}
					}
				}

			case err := <-errChan:
				if err != nil {
					errorChan <- fmt.Errorf("streaming error during task execution: %v", err)
					return
				}

			case <-ctx.Done():
				errorChan <- fmt.Errorf("context cancelled during task execution")
				return
			}
		}
	}()

	return resultChan, errorChan
}

// Helper functions for pointer creation (required by the SDK)
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func outputFormatPtr(f claudecode.OutputFormat) *claudecode.OutputFormat {
	return &f
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
