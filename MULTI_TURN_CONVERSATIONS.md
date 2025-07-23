# Multi-Turn Conversations for Task Execution

This document outlines the implementation plan and usage of the multi-turn conversation system that enables iterative task development with Claude Code.

## Overview

The multi-turn conversation system allows you to:

1. **Start a conversation** for a task and get initial implementation
2. **Continue the conversation** with modifications, refinements, or additional requests
3. **Maintain context** across multiple interactions using Claude Code's session management
4. **Track changes** and manage Git commits throughout the conversation
5. **Resume conversations** later using session IDs

## Key Components

### 1. Data Structures

#### TaskConversation
```go
type TaskConversation struct {
    TaskID           int                     `json:"taskId"`
    WorkspaceName    string                  `json:"workspaceName"`
    BranchName       string                  `json:"branchName"`
    WorktreePath     string                  `json:"worktreePath"`
    ConversationID   string                  `json:"conversationId"`
    IsActive         bool                    `json:"isActive"`
    CreatedAt        time.Time               `json:"createdAt"`
    LastMessageAt    time.Time               `json:"lastMessageAt"`
    Messages         []ConversationMessage   `json:"messages"`
    Context          ConversationContext     `json:"context"`
}
```

#### ConversationMessage
```go
type ConversationMessage struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"` // "user", "assistant", "system"
    Content   string                 `json:"content"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

#### ConversationContext
```go
type ConversationContext struct {
    TaskTitle       string   `json:"taskTitle"`
    TaskDescription string   `json:"taskDescription"`
    BaseBranch     string   `json:"baseBranch"`
    FilesChanged   []string `json:"filesChanged"`
    LastCommitHash string   `json:"lastCommitHash,omitempty"`
    SessionID      string   `json:"sessionId,omitempty"`
}
```

### 2. Claude Client Enhancements

#### ContinueConversation Method
Uses Claude Code's `Resume` option to continue a specific session:
```go
func (c *ClaudeClient) ContinueConversation(sessionId, userMessage string) TaskExecutionResult {
    request := claudecode.QueryRequest{
        Prompt: userMessage,
        Options: &claudecode.Options{
            Resume: &sessionId, // Resume existing session
            // ... other options
        },
    }
    // ... implementation
}
```

#### ContinueLatestSession Method
Uses Claude Code's `Continue` option to continue the most recent session:
```go
func (c *ClaudeClient) ContinueLatestSession(userMessage string) TaskExecutionResult {
    request := claudecode.QueryRequest{
        Prompt: userMessage,
        Options: &claudecode.Options{
            Continue: boolPtr(true), // Continue latest session
            // ... other options
        },
    }
    // ... implementation
}
```

### 3. Application Methods

#### StartTaskConversation
- Creates Git worktree for task isolation
- Starts initial Claude Code session
- Saves conversation state
- Returns conversation ID for future use

#### ContinueTaskConversation
- Loads existing conversation
- Continues Claude Code session using session ID
- Updates conversation with new messages
- Commits changes if any files were modified

#### GetTaskConversations
- Lists all conversations for a workspace or specific task
- Filters by active/inactive status

#### EndTaskConversation
- Marks conversation as inactive
- Preserves conversation history

## Usage Flow

### 1. Starting a Conversation

```typescript
// Frontend calls this to start a task conversation
const result = await StartTaskConversation(
    "my-workspace",
    42,
    "Implement user authentication",
    "Add JWT-based authentication with login/logout",
    "main"
);

console.log(result.conversationId); // "conv-42-my-workspace-1704067200"
```

### 2. Continuing the Conversation

```typescript
// User provides feedback or additional requirements
const continueResult = await ContinueTaskConversation(
    "conv-42-my-workspace-1704067200",
    "Please add password validation and make the login form more responsive"
);

console.log(continueResult.response); // Claude's response with additional changes
```

### 3. Multiple Iterations

```typescript
// Keep refining until satisfied
await ContinueTaskConversation(
    "conv-42-my-workspace-1704067200",
    "Add unit tests for the authentication logic"
);

await ContinueTaskConversation(
    "conv-42-my-workspace-1704067200",
    "Make the error messages more user-friendly"
);
```

### 4. Ending the Conversation

```typescript
// When done with the task
await EndTaskConversation("conv-42-my-workspace-1704067200");
```

## Benefits

### 1. **Iterative Development**
- Make incremental improvements without starting from scratch
- Refine implementations based on testing or feedback
- Add features progressively

### 2. **Context Preservation**
- Claude remembers previous changes and decisions
- Maintains awareness of the codebase state
- Builds upon previous implementations

### 3. **Git Integration**
- Each conversation step can create commits
- Changes are isolated in task-specific worktrees
- Full Git history of the development process

### 4. **Session Management**
- Uses Claude Code's native session system
- Automatically resumes conversations
- Handles session state persistence

## File Storage

### Conversation Files
Stored in `~/.aicodingtool/conversations/{conversationID}.json`

Example conversation file:
```json
{
  "taskId": 42,
  "workspaceName": "my-workspace",
  "branchName": "task-42-implement-user-auth",
  "worktreePath": "/path/to/task-42-my-workspace",
  "conversationId": "conv-42-my-workspace-1704067200",
  "isActive": true,
  "createdAt": "2024-01-01T12:00:00Z",
  "lastMessageAt": "2024-01-01T12:30:00Z",
  "messages": [
    {
      "id": "msg-1704067200000",
      "type": "system",
      "content": "Started conversation for task 42: Implement user authentication",
      "timestamp": "2024-01-01T12:00:00Z"
    },
    {
      "id": "msg-1704067200001",
      "type": "assistant",
      "content": "I've implemented JWT-based authentication...",
      "timestamp": "2024-01-01T12:00:30Z",
      "metadata": {
        "filesChanged": ["auth/jwt.go", "handlers/login.go"],
        "sessionId": "session-abc123"
      }
    }
  ],
  "context": {
    "taskTitle": "Implement user authentication",
    "taskDescription": "Add JWT-based authentication with login/logout",
    "baseBranch": "main",
    "filesChanged": ["auth/jwt.go", "handlers/login.go"],
    "sessionId": "session-abc123"
  }
}
```

## Frontend Integration

The frontend would need new UI components:

### 1. **Conversation Mode Toggle**
When running a task, offer option to "Start Conversation" vs "Run Once"

### 2. **Chat Interface**
For active conversations, show:
- Previous messages
- Input field for new requests
- File changes for each step
- Option to end conversation

### 3. **Conversation History**
List previous conversations for each task with ability to resume

## Error Handling

### Session Recovery
If a session ID becomes invalid:
1. Try using `ContinueLatestSession` as fallback
2. If that fails, offer to start new conversation
3. Preserve conversation history

### Worktree Cleanup
- Automatically clean up worktrees for ended conversations
- Provide manual cleanup options
- Handle orphaned worktrees

## Security Considerations

### 1. **Conversation Isolation**
- Each conversation uses isolated Git worktree
- No cross-contamination between tasks
- Proper cleanup prevents disk space issues

### 2. **Session Security**
- Session IDs are stored locally only
- No sensitive data in conversation files
- Standard Git security practices apply

## Future Enhancements

### 1. **Conversation Branching**
- Fork conversations to try different approaches
- Compare results from different conversation paths

### 2. **Conversation Templates**
- Save common conversation patterns
- Quick-start conversations for typical tasks

### 3. **Collaboration**
- Share conversation IDs with team members
- Merge conversations from multiple developers

### 4. **Analytics**
- Track conversation success rates
- Identify common patterns
- Optimize conversation strategies

## Implementation Status

✅ **Completed:**
- Data structures defined
- Claude client methods implemented
- Basic conversation management
- Session management with Claude Code SDK

🚧 **In Progress:**
- Frontend UI components
- Error handling refinements
- Session recovery mechanisms

📋 **Planned:**
- Conversation templates
- Advanced collaboration features
- Performance optimizations

This multi-turn conversation system transforms the one-shot task execution into an interactive development experience, allowing for iterative refinement and much more sophisticated task completion. 