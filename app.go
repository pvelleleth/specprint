package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"specprint/pkg/claude"

	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/sashabaranov/go-openai"
)

// App struct
type App struct {
	ctx context.Context
}

// CloneResult represents the result of a repository clone operation
type CloneResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// PRDResult represents the result of a PRD save operation
type PRDResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// ClaudeSessionResult represents the result of Claude operations
type ClaudeSessionResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	Response     string   `json:"response,omitempty"`
	FilesChanged []string `json:"filesChanged,omitempty"`
}

// Task represents a single implementation task
type Task struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Dependencies []int  `json:"dependencies"`
	Priority     string `json:"priority"` // "high", "medium", "low"
	Estimate     string `json:"estimate"` // e.g., "2h", "1d", "3d"
}

// TaskGenerationResult represents the result of task generation
type TaskGenerationResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Tasks   []Task `json:"tasks,omitempty"` // Changed from Epics []Epic
}

// Workspace represents a cloned repository workspace
type Workspace struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	RepoURL    string    `json:"repoUrl"`
	ClonedAt   time.Time `json:"clonedAt"`
	LastOpened time.Time `json:"lastOpened"`
	HasPRD     bool      `json:"hasPrd"`
	PRDPath    string    `json:"prdPath,omitempty"`
}

// WorkspacesResult represents the result of listing workspaces
type WorkspacesResult struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Workspaces []Workspace `json:"workspaces,omitempty"`
}

// TaskExecutionResult represents the result of executing a task with Git branching and Claude
type TaskExecutionResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	BranchName   string   `json:"branchName,omitempty"`
	FilesChanged []string `json:"filesChanged,omitempty"`
	ClaudeOutput string   `json:"claudeOutput,omitempty"`
	SessionID    string   `json:"sessionId,omitempty"`
	WorktreePath string   `json:"worktreePath,omitempty"`
}

// BranchInfo represents information about a Git branch
type BranchInfo struct {
	Name      string `json:"name"`
	IsRemote  bool   `json:"isRemote"`
	IsCurrent bool   `json:"isCurrent"`
	Hash      string `json:"hash,omitempty"`
}

// BranchListResult represents the result of listing branches
type BranchListResult struct {
	Success  bool         `json:"success"`
	Message  string       `json:"message"`
	Branches []BranchInfo `json:"branches,omitempty"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Load environment variables from .env file if it exists
	loadEnvFile()
	return &App{}
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		return // .env file doesn't exist, which is fine
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}

			// Only set if not already set in environment
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GenerateTasks uses OpenAI to parse PRD content and generate structured tasks
func (a *App) GenerateTasks(prdContent string) TaskGenerationResult {
	// Validate input
	if strings.TrimSpace(prdContent) == "" {
		return TaskGenerationResult{
			Success: false,
			Message: "PRD content cannot be empty",
		}
	}

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return TaskGenerationResult{
			Success: false,
			Message: "OPENAI_API_KEY environment variable is not set",
		}
	}

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	// Construct the system prompt
	systemPrompt := `You are an expert project manager and software architect. Your task is to analyze a Product Requirements Document (PRD) and generate a flat list of actionable tasks with proper dependencies.

STRUCTURE:
- TASKS: Specific implementation tasks (aim for 20-50 tasks total, depending on PRD complexity)

For each task, provide:
- id: Unique sequential number starting from 1
- title: Specific task name (max 80 characters)
- description: What needs to be done (max 200 characters)
- dependencies: Array of task IDs that must be completed first (use [] if none)
- priority: "high", "medium", or "low"
- estimate: Time estimate like "2h", "1d", "3d"

DEPENDENCY RULES:
1. **Setup Dependencies**: Infrastructure and setup tasks should have no dependencies
2. **Logical Sequencing**: Tasks that depend on other tasks' outputs should reference them
3. **Phase Dependencies**: Implementation tasks should depend on design tasks
4. **Integration Dependencies**: API integration tasks should depend on backend tasks
5. **Testing Dependencies**: Test tasks should depend on the features they test
6. **Deployment Dependencies**: Deployment tasks should depend on all implementation tasks

COMMON DEPENDENCY PATTERNS:
- Database setup → Backend API → Frontend integration → Testing → Deployment
- Design system → UI components → Feature implementation → Integration testing
- Authentication setup → User management → Protected features → Security testing
- API design → Backend implementation → Frontend API calls → End-to-end testing

TASK GENERATION RULES:
1. Break down the PRD into specific, actionable tasks
2. Consider dependencies and sequencing carefully
3. Include setup, implementation, testing, and deployment phases
4. Provide realistic time estimates
5. Cover all aspects of the PRD comprehensively
6. Ensure tasks are independent but properly sequenced via dependencies
7. Create logical task groups that can be worked on in parallel when possible

Return ONLY a valid JSON array of tasks. Do not include any other text or formatting.

Example format:
[
  {
    "id": 1,
    "title": "Set up project repository",
    "description": "Initialize Git repository and basic structure",
    "dependencies": [],
    "priority": "high",
    "estimate": "1h"
  },
  {
    "id": 2,
    "title": "Design database schema",
    "description": "Create database tables and relationships",
    "dependencies": [],
    "priority": "high",
    "estimate": "4h"
  },
  {
    "id": 3,
    "title": "Implement user authentication",
    "description": "Create login/register API endpoints",
    "dependencies": [1, 2],
    "priority": "high",
    "estimate": "8h"
  }
]`

	// Create the chat completion request
	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("Please analyze this PRD and generate implementation tasks:\n\n%s", prdContent),
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1, // Low temperature for consistent, structured output
	}

	// Make the API call
	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return TaskGenerationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to call OpenAI API: %v", err),
		}
	}

	if len(resp.Choices) == 0 {
		return TaskGenerationResult{
			Success: false,
			Message: "No response received from OpenAI",
		}
	}

	// Get the response content
	responseContent := resp.Choices[0].Message.Content

	// Parse the JSON response
	var tasks []Task // Changed from var epics []Epic
	err = json.Unmarshal([]byte(responseContent), &tasks)
	if err != nil {
		return TaskGenerationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse JSON response: %v. Response was: %s", err, responseContent),
		}
	}

	// Validate the parsed epics
	if len(tasks) == 0 {
		return TaskGenerationResult{
			Success: false,
			Message: "No tasks were generated from the PRD",
		}
	}

	for i, task := range tasks {
		if task.ID <= 0 {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Task %d has invalid ID: %d", i+1, task.ID),
			}
		}
		if strings.TrimSpace(task.Title) == "" {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Task %d has empty title", task.ID),
			}
		}
		if strings.TrimSpace(task.Description) == "" {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Task %d has empty description", task.ID),
			}
		}
		if strings.TrimSpace(task.Priority) == "" {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Task %d has empty priority", task.ID),
			}
		}
		if strings.TrimSpace(task.Estimate) == "" {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Task %d has empty estimate", task.ID),
			}
		}
		if task.Dependencies == nil {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Task %d has nil dependencies", task.ID),
			}
		}
	}

	return TaskGenerationResult{
		Success: true,
		Message: fmt.Sprintf("Successfully generated %d tasks from PRD", len(tasks)),
		Tasks:   tasks, // Changed from Epics: epics
	}
}

// GenerateTasksFromWorkspacePRD generates tasks from a specific workspace's PRD file
func (a *App) GenerateTasksFromWorkspacePRD(workspaceName string) TaskGenerationResult {
	// Validate workspace name
	if strings.TrimSpace(workspaceName) == "" {
		return TaskGenerationResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	// Get workspaces
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return TaskGenerationResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return TaskGenerationResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Check if PRD exists
	if !targetWorkspace.HasPRD {
		return TaskGenerationResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' does not have a PRD file", workspaceName),
		}
	}

	// Read PRD content
	prdContent, err := os.ReadFile(targetWorkspace.PRDPath)
	if err != nil {
		return TaskGenerationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to read PRD file: %v", err),
		}
	}

	// Generate tasks using the PRD content
	return a.GenerateTasks(string(prdContent)) // Now returns tasks
}

// GetWorkspaces returns all available workspaces
func (a *App) GetWorkspaces() WorkspacesResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return WorkspacesResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get user home directory: %v", err),
		}
	}

	baseDir := filepath.Join(homeDir, ".aicodingtool", "repos")
	workspacesFile := filepath.Join(homeDir, ".aicodingtool", "workspaces.json")

	// Check if workspaces file exists
	var workspaces []Workspace
	if _, err := os.Stat(workspacesFile); err == nil {
		data, err := os.ReadFile(workspacesFile)
		if err == nil {
			json.Unmarshal(data, &workspaces)
		}
	}

	// Update workspace info from filesystem
	if _, err := os.Stat(baseDir); err == nil {
		repos, err := os.ReadDir(baseDir)
		if err == nil {
			// First pass: clean up any orphaned worktree directories
			for _, repo := range repos {
				if repo.IsDir() && a.isWorktreeDirectory(repo.Name()) {
					// This is a worktree directory - check if it should be cleaned up
					repoPath := filepath.Join(baseDir, repo.Name())
					a.checkAndCleanupOrphanedWorktree(repoPath, repo.Name())
				}
			}

			// Second pass: process actual workspaces
			for _, repo := range repos {
				if repo.IsDir() {
					repoPath := filepath.Join(baseDir, repo.Name())

					// Skip worktree directories (they follow the pattern task-{number}-{workspacename})
					if a.isWorktreeDirectory(repo.Name()) {
						continue
					}

					// Check if this workspace already exists in our list
					found := false
					for i := range workspaces {
						if workspaces[i].Name == repo.Name() {
							// Update existing workspace
							workspaces[i].Path = repoPath
							workspaces[i].HasPRD = a.checkPRDExists(repoPath)
							if workspaces[i].HasPRD {
								workspaces[i].PRDPath = filepath.Join(repoPath, "PRD.md")
							}
							found = true
							break
						}
					}

					// If not found, add as new workspace
					if !found {
						info, _ := repo.Info()
						workspace := Workspace{
							Name:       repo.Name(),
							Path:       repoPath,
							ClonedAt:   info.ModTime(),
							LastOpened: info.ModTime(),
							HasPRD:     a.checkPRDExists(repoPath),
						}
						if workspace.HasPRD {
							workspace.PRDPath = filepath.Join(repoPath, "PRD.md")
						}
						workspaces = append(workspaces, workspace)
					}
				}
			}
		}
	}

	// Remove duplicates before saving
	workspaces = a.deduplicateWorkspaces(workspaces)

	// Save updated workspaces
	a.saveWorkspaces(workspaces)

	return WorkspacesResult{
		Success:    true,
		Message:    fmt.Sprintf("Found %d workspaces", len(workspaces)),
		Workspaces: workspaces,
	}
}

// SaveWorkspacePRD saves PRD content to a specific workspace
func (a *App) SaveWorkspacePRD(workspaceName, prdContent string) PRDResult {
	// Validate content
	if strings.TrimSpace(prdContent) == "" {
		return PRDResult{
			Success: false,
			Message: "PRD content cannot be empty",
		}
	}

	if strings.TrimSpace(workspaceName) == "" {
		return PRDResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	// Get workspaces
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return PRDResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return PRDResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Create PRD.md file in the workspace
	prdFilePath := filepath.Join(targetWorkspace.Path, "PRD.md")

	// Add timestamp to the PRD content
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prdWithTimestamp := fmt.Sprintf("# Product Requirements Document\n\n*Generated on: %s*\n*Workspace: %s*\n\n---\n\n%s", timestamp, workspaceName, prdContent)

	// Write the PRD content to the file
	err := os.WriteFile(prdFilePath, []byte(prdWithTimestamp), 0644)
	if err != nil {
		return PRDResult{
			Success: false,
			Message: fmt.Sprintf("Failed to write PRD file: %v", err),
		}
	}

	// Update workspace with PRD info
	targetWorkspace.HasPRD = true
	targetWorkspace.PRDPath = prdFilePath
	targetWorkspace.LastOpened = time.Now()

	// Save updated workspaces
	a.saveWorkspaces(workspacesResult.Workspaces)

	return PRDResult{
		Success: true,
		Message: fmt.Sprintf("PRD saved successfully to workspace: %s", workspaceName),
		Path:    prdFilePath,
	}
}

// OpenWorkspace updates the last opened time for a workspace
func (a *App) OpenWorkspace(workspaceName string) PRDResult {
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return PRDResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find and update the workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			workspacesResult.Workspaces[i].LastOpened = time.Now()
			a.saveWorkspaces(workspacesResult.Workspaces)
			return PRDResult{
				Success: true,
				Message: fmt.Sprintf("Opened workspace: %s", workspaceName),
			}
		}
	}

	return PRDResult{
		Success: false,
		Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
	}
}

// checkPRDExists checks if PRD.md exists in the workspace
func (a *App) checkPRDExists(workspacePath string) bool {
	prdPath := filepath.Join(workspacePath, "PRD.md")
	_, err := os.Stat(prdPath)
	return err == nil
}

// saveWorkspaces saves the workspaces to the JSON file
func (a *App) saveWorkspaces(workspaces []Workspace) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	aiToolDir := filepath.Join(homeDir, ".aicodingtool")
	if err := os.MkdirAll(aiToolDir, 0755); err != nil {
		return err
	}

	workspacesFile := filepath.Join(aiToolDir, "workspaces.json")
	data, err := json.MarshalIndent(workspaces, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(workspacesFile, data, 0644)
}

// deduplicateWorkspaces removes duplicate workspaces based on name, keeping the most recent one
func (a *App) deduplicateWorkspaces(workspaces []Workspace) []Workspace {
	seen := make(map[string]int) // name -> index of most recent
	var result []Workspace

	for _, workspace := range workspaces {
		if existingIndex, exists := seen[workspace.Name]; exists {
			// Compare LastOpened times and keep the more recent one
			if workspace.LastOpened.After(result[existingIndex].LastOpened) {
				result[existingIndex] = workspace
			}
		} else {
			seen[workspace.Name] = len(result)
			result = append(result, workspace)
		}
	}

	return result
}

// CleanupDuplicateWorkspaces removes duplicate workspaces from the system
func (a *App) CleanupDuplicateWorkspaces() DeleteWorkspaceResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return DeleteWorkspaceResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get user home directory: %v", err),
		}
	}

	workspacesFile := filepath.Join(homeDir, ".aicodingtool", "workspaces.json")

	// Load existing workspaces
	var workspaces []Workspace
	if _, err := os.Stat(workspacesFile); err == nil {
		data, err := os.ReadFile(workspacesFile)
		if err != nil {
			return DeleteWorkspaceResult{
				Success: false,
				Message: fmt.Sprintf("Failed to read workspaces file: %v", err),
			}
		}
		if err := json.Unmarshal(data, &workspaces); err != nil {
			return DeleteWorkspaceResult{
				Success: false,
				Message: fmt.Sprintf("Failed to parse workspaces file: %v", err),
			}
		}
	}

	originalCount := len(workspaces)
	workspaces = a.deduplicateWorkspaces(workspaces)
	newCount := len(workspaces)
	duplicatesRemoved := originalCount - newCount

	// Save cleaned workspaces
	if err := a.saveWorkspaces(workspaces); err != nil {
		return DeleteWorkspaceResult{
			Success: false,
			Message: fmt.Sprintf("Failed to save cleaned workspaces: %v", err),
		}
	}

	return DeleteWorkspaceResult{
		Success: true,
		Message: fmt.Sprintf("Removed %d duplicate workspace(s). %d workspaces remaining.", duplicatesRemoved, newCount),
	}
}

// DeleteWorkspaceResult represents the result of deleting a workspace
type DeleteWorkspaceResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeleteWorkspace removes a workspace from the system
func (a *App) DeleteWorkspace(workspaceName string, deleteFiles bool) DeleteWorkspaceResult {
	// Validate input
	if strings.TrimSpace(workspaceName) == "" {
		return DeleteWorkspaceResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	// Get current workspaces
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return DeleteWorkspaceResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get workspaces: %s", workspacesResult.Message),
		}
	}

	// Find the workspace to delete
	var targetWorkspace *Workspace
	var workspaceIndex int
	for i, workspace := range workspacesResult.Workspaces {
		if workspace.Name == workspaceName {
			targetWorkspace = &workspace
			workspaceIndex = i
			break
		}
	}

	if targetWorkspace == nil {
		return DeleteWorkspaceResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Clean up any active worktrees for this workspace
	a.cleanupAllWorktrees(targetWorkspace.Path, workspaceName)

	// Remove workspace from the list
	updatedWorkspaces := make([]Workspace, 0, len(workspacesResult.Workspaces)-1)
	for i, workspace := range workspacesResult.Workspaces {
		if i != workspaceIndex {
			updatedWorkspaces = append(updatedWorkspaces, workspace)
		}
	}

	// Save updated workspaces list
	if err := a.saveWorkspaces(updatedWorkspaces); err != nil {
		return DeleteWorkspaceResult{
			Success: false,
			Message: fmt.Sprintf("Failed to update workspaces file: %v", err),
		}
	}

	// Optionally delete the physical files
	if deleteFiles {
		if err := os.RemoveAll(targetWorkspace.Path); err != nil {
			return DeleteWorkspaceResult{
				Success: false,
				Message: fmt.Sprintf("Workspace removed from list but failed to delete files at '%s': %v", targetWorkspace.Path, err),
			}
		}
		return DeleteWorkspaceResult{
			Success: true,
			Message: fmt.Sprintf("Successfully deleted workspace '%s' and all its files", workspaceName),
		}
	}

	return DeleteWorkspaceResult{
		Success: true,
		Message: fmt.Sprintf("Successfully removed workspace '%s' from the list (files preserved at '%s')", workspaceName, targetWorkspace.Path),
	}
}

// cleanupAllWorktrees removes all worktrees associated with a workspace
func (a *App) cleanupAllWorktrees(workspacePath, workspaceName string) {
	// Get the parent directory where worktrees would be created
	baseDir := filepath.Dir(workspacePath)

	// Look for worktree directories that match the pattern task-*-workspaceName
	pattern := fmt.Sprintf("task-*-%s", workspaceName)
	matches, err := filepath.Glob(filepath.Join(baseDir, pattern))
	if err != nil {
		fmt.Printf("Warning: Failed to find worktrees for workspace %s: %v\n", workspaceName, err)
		return
	}

	// Remove each worktree
	for _, worktreePath := range matches {
		fmt.Printf("Cleaning up worktree: %s\n", worktreePath)

		// First try to remove the worktree using git command
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
		cmd.Dir = workspacePath
		if err := cmd.Run(); err != nil {
			// If git worktree remove fails, just delete the directory
			fmt.Printf("Git worktree remove failed, deleting directory: %v\n", err)
		}

		// Ensure the directory is gone
		if _, err := os.Stat(worktreePath); err == nil {
			os.RemoveAll(worktreePath)
		}
	}
}

// SavePRD saves the PRD content to a file in the repository (deprecated - use SaveWorkspacePRD)
func (a *App) SavePRD(prdContent string) PRDResult {
	// Validate content
	if strings.TrimSpace(prdContent) == "" {
		return PRDResult{
			Success: false,
			Message: "PRD content cannot be empty",
		}
	}

	// Get the base directory for repositories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return PRDResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get user home directory: %v", err),
		}
	}

	baseDir := filepath.Join(homeDir, ".aicodingtool", "repos")

	// Check if the base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return PRDResult{
			Success: false,
			Message: "No repositories found. Please clone a repository first.",
		}
	}

	// Find the most recently cloned repository
	repos, err := os.ReadDir(baseDir)
	if err != nil {
		return PRDResult{
			Success: false,
			Message: fmt.Sprintf("Failed to read repositories directory: %v", err),
		}
	}

	if len(repos) == 0 {
		return PRDResult{
			Success: false,
			Message: "No repositories found. Please clone a repository first.",
		}
	}

	// For now, we'll use the first repository found
	// In a more sophisticated implementation, you might want to let the user choose
	repoName := repos[0].Name()
	repoPath := filepath.Join(baseDir, repoName)

	// Create PRD.md file in the repository
	prdFilePath := filepath.Join(repoPath, "PRD.md")

	// Add timestamp to the PRD content
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prdWithTimestamp := fmt.Sprintf("# Product Requirements Document\n\n*Generated on: %s*\n\n---\n\n%s", timestamp, prdContent)

	// Write the PRD content to the file
	err = os.WriteFile(prdFilePath, []byte(prdWithTimestamp), 0644)
	if err != nil {
		return PRDResult{
			Success: false,
			Message: fmt.Sprintf("Failed to write PRD file: %v", err),
		}
	}

	return PRDResult{
		Success: true,
		Message: fmt.Sprintf("PRD saved successfully to repository: %s", repoName),
		Path:    prdFilePath,
	}
}

// CloneRepository clones a Git repository into the dedicated app directory
func (a *App) CloneRepository(repoURL string) CloneResult {
	// Validate URL format
	if !isValidGitURL(repoURL) {
		return CloneResult{
			Success: false,
			Message: "Invalid Git repository URL. Please provide a valid HTTPS or SSH URL.",
		}
	}

	// Create the base directory for repositories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CloneResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get user home directory: %v", err),
		}
	}

	baseDir := filepath.Join(homeDir, ".aicodingtool", "repos")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return CloneResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create repository directory: %v", err),
		}
	}

	// Extract repository name from URL
	repoName := extractRepoName(repoURL)
	if repoName == "" {
		return CloneResult{
			Success: false,
			Message: "Could not extract repository name from URL",
		}
	}

	// Create the target directory
	targetDir := filepath.Join(baseDir, repoName)

	// Check if directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		return CloneResult{
			Success: false,
			Message: fmt.Sprintf("Repository directory already exists: %s", targetDir),
		}
	}

	// Clone the repository
	repo, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})

	if err != nil {
		return CloneResult{
			Success: false,
			Message: fmt.Sprintf("Failed to clone repository: %v", err),
		}
	}

	// Verify the repository was cloned successfully
	head, err := repo.Head()
	if err != nil {
		return CloneResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get repository head: %v", err),
		}
	}

	// Add workspace to the list
	workspace := Workspace{
		Name:       repoName,
		Path:       targetDir,
		RepoURL:    repoURL,
		ClonedAt:   time.Now(),
		LastOpened: time.Now(),
		HasPRD:     a.checkPRDExists(targetDir),
	}
	if workspace.HasPRD {
		workspace.PRDPath = filepath.Join(targetDir, "PRD.md")
	}

	// Get existing workspaces directly from file (without filesystem scan)
	workspacesFile := filepath.Join(homeDir, ".aicodingtool", "workspaces.json")

	var workspaces []Workspace
	if _, err := os.Stat(workspacesFile); err == nil {
		data, err := os.ReadFile(workspacesFile)
		if err == nil {
			json.Unmarshal(data, &workspaces)
		}
	}

	// Check if workspace already exists
	found := false
	for i, existingWorkspace := range workspaces {
		if existingWorkspace.Name == repoName {
			// Update existing workspace instead of creating duplicate
			workspaces[i] = workspace
			found = true
			break
		}
	}

	// Only add if not found
	if !found {
		workspaces = append(workspaces, workspace)
	}

	a.saveWorkspaces(workspaces)

	return CloneResult{
		Success: true,
		Message: fmt.Sprintf("Successfully cloned repository. Current branch: %s", head.Name().Short()),
		Path:    targetDir,
	}
}

// isValidGitURL validates if the provided URL is a valid Git repository URL
func isValidGitURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return false
	}

	// Check for HTTPS URLs
	if strings.HasPrefix(url, "https://") {
		return true
	}

	// Check for SSH URLs (git@github.com:user/repo.git)
	if strings.HasPrefix(url, "git@") && strings.Contains(url, ":") {
		return true
	}

	// Check for SSH URLs with ssh:// prefix
	if strings.HasPrefix(url, "ssh://") {
		return true
	}

	return false
}

// isWorktreeDirectory checks if a directory name follows the worktree pattern (task-{number}-{workspacename})
func (a *App) isWorktreeDirectory(dirName string) bool {
	// Worktree directories follow the pattern: task-{number}-{workspacename}
	// Example: task-1-myproject, task-42-frontend-app
	if !strings.HasPrefix(dirName, "task-") {
		return false
	}

	// Split by hyphens and check if it has at least 3 parts: "task", number, workspace
	parts := strings.Split(dirName, "-")
	if len(parts) < 3 {
		return false
	}

	// Check if the second part is a number (task ID)
	if _, err := strconv.Atoi(parts[1]); err != nil {
		return false
	}

	// If we get here, it matches the worktree pattern
	return true
}

// checkAndCleanupOrphanedWorktree removes worktree directories that may be left over
func (a *App) checkAndCleanupOrphanedWorktree(worktreePath, dirName string) {
	// Extract workspace name from directory name (task-{number}-{workspacename})
	parts := strings.Split(dirName, "-")
	if len(parts) < 3 {
		return
	}

	// The workspace name is everything after "task-{number}-"
	workspaceName := strings.Join(parts[2:], "-")

	// Check if the corresponding main workspace exists
	baseDir := filepath.Dir(worktreePath)
	mainWorkspacePath := filepath.Join(baseDir, workspaceName)

	if _, err := os.Stat(mainWorkspacePath); os.IsNotExist(err) {
		// Main workspace doesn't exist, this worktree is orphaned
		fmt.Printf("Cleaning up orphaned worktree: %s (main workspace '%s' not found)\n", worktreePath, workspaceName)
		os.RemoveAll(worktreePath)
	}
}

// extractRepoName extracts the repository name from a Git URL
func extractRepoName(url string) string {
	url = strings.TrimSpace(url)

	// Handle HTTPS URLs: https://github.com/user/repo.git
	if strings.HasPrefix(url, "https://") {
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			lastPart := parts[len(parts)-1]
			return strings.TrimSuffix(lastPart, ".git")
		}
	}

	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) >= 2 {
			lastPart := parts[len(parts)-1]
			return strings.TrimSuffix(lastPart, ".git")
		}
	}

	// Handle SSH URLs with ssh:// prefix: ssh://git@github.com/user/repo.git
	if strings.HasPrefix(url, "ssh://") {
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			lastPart := parts[len(parts)-1]
			return strings.TrimSuffix(lastPart, ".git")
		}
	}

	return ""
}

// GetWorkspaceBranches retrieves all available branches for a workspace
func (a *App) GetWorkspaceBranches(workspaceName string) BranchListResult {
	// Validate workspace name
	if strings.TrimSpace(workspaceName) == "" {
		return BranchListResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	// Get workspaces to find the target workspace
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return BranchListResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Open the Git repository
	repo, err := git.PlainOpen(targetWorkspace.Path)
	if err != nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Failed to open Git repository: %v", err),
		}
	}

	// Get current branch
	currentHead, err := repo.Head()
	if err != nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get repository HEAD: %v", err),
		}
	}
	currentBranchName := currentHead.Name().Short()

	var branches []BranchInfo

	// Get local branches
	branchIter, err := repo.Branches()
	if err != nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get local branches: %v", err),
		}
	}

	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		branchName := ref.Name().Short()
		branches = append(branches, BranchInfo{
			Name:      branchName,
			IsRemote:  false,
			IsCurrent: branchName == currentBranchName,
			Hash:      ref.Hash().String()[:8], // Short hash
		})
		return nil
	})
	if err != nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Failed to iterate local branches: %v", err),
		}
	}

	// Get remote branches
	remoteIter, err := repo.References()
	if err != nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get remote branches: %v", err),
		}
	}

	err = remoteIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			// Extract branch name from refs/remotes/origin/branch-name
			fullName := ref.Name().String()
			if strings.HasPrefix(fullName, "refs/remotes/origin/") && !strings.HasSuffix(fullName, "/HEAD") {
				branchName := strings.TrimPrefix(fullName, "refs/remotes/origin/")

				// Check if we already have this branch locally
				hasLocal := false
				for _, localBranch := range branches {
					if localBranch.Name == branchName {
						hasLocal = true
						break
					}
				}

				// Only add if not present locally
				if !hasLocal {
					branches = append(branches, BranchInfo{
						Name:      branchName,
						IsRemote:  true,
						IsCurrent: false,
						Hash:      ref.Hash().String()[:8], // Short hash
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return BranchListResult{
			Success: false,
			Message: fmt.Sprintf("Failed to iterate remote branches: %v", err),
		}
	}

	return BranchListResult{
		Success:  true,
		Message:  fmt.Sprintf("Found %d branches for workspace '%s'", len(branches), workspaceName),
		Branches: branches,
	}
}

// RunTask executes a task by creating a Git branch and running Claude Code
func (a *App) RunTask(workspaceName string, taskID int, taskTitle, taskDescription, baseBranch string) TaskExecutionResult {
	// Validate input parameters
	if strings.TrimSpace(workspaceName) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	if taskID <= 0 {
		return TaskExecutionResult{
			Success: false,
			Message: "Task ID must be a positive integer",
		}
	}

	if strings.TrimSpace(taskTitle) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Task title cannot be empty",
		}
	}

	if strings.TrimSpace(baseBranch) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Base branch cannot be empty",
		}
	}

	// Get workspaces to find the target workspace
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return TaskExecutionResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Open the main Git repository
	repo, err := git.PlainOpen(targetWorkspace.Path)
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to open Git repository: %v", err),
		}
	}

	// Step 1: Fetch from origin in main repository
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch from origin: %v", err),
		}
	}

	// Step 2: Generate branch name and worktree path
	branchName := generateBranchName(taskID, taskTitle)
	worktreePath := filepath.Join(filepath.Dir(targetWorkspace.Path), fmt.Sprintf("task-%d-%s", taskID, workspaceName))

	// Clean up any existing worktree directory (in case of previous failure)
	if _, err := os.Stat(worktreePath); err == nil {
		os.RemoveAll(worktreePath)
	}

	// Step 3: Create worktree from base branch
	// First ensure the base branch reference exists
	baseBranchRef := plumbing.NewBranchReferenceName(baseBranch)
	_, err = repo.Reference(baseBranchRef, true)
	if err != nil {
		// Try to find remote branch
		remoteBranchRef := plumbing.NewRemoteReferenceName("origin", baseBranch)
		remoteRef, err := repo.Reference(remoteBranchRef, true)
		if err != nil {
			return TaskExecutionResult{
				Success: false,
				Message: fmt.Sprintf("Base branch '%s' not found locally or remotely: %v", baseBranch, err),
			}
		}

		// Create local branch from remote
		localRef := plumbing.NewHashReference(baseBranchRef, remoteRef.Hash())
		err = repo.Storer.SetReference(localRef)
		if err != nil {
			return TaskExecutionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to create local branch '%s': %v", baseBranch, err),
			}
		}
	}

	// Step 4: Create git worktree using command line (go-git doesn't support worktrees directly)
	result := a.executeGitWorktreeCommands(targetWorkspace.Path, worktreePath, baseBranch, branchName)
	if !result.Success {
		return result
	}

	// Step 5: Initialize Claude client with the worktree path
	claudeClient := claude.NewClaudeClient(worktreePath)

	// Execute the task using Claude Code in the worktree
	claudeResult := claudeClient.ExecuteTask(taskID, taskTitle, taskDescription)
	if !claudeResult.Success {
		return TaskExecutionResult{
			Success:    false,
			Message:    fmt.Sprintf("Claude Code execution failed: %s", claudeResult.Message),
			BranchName: branchName,
		}
	}

	// Step 6: Check for any changes in the worktree and commit/push if found
	hasChanges, changedFiles := a.checkForGitChanges(worktreePath)
	if hasChanges {
		// Use detected files if Claude didn't report any, otherwise use Claude's list
		filesToCommit := claudeResult.FilesChanged
		if len(filesToCommit) == 0 {
			filesToCommit = changedFiles
		}

		commitResult := a.commitAndPushFromWorktree(worktreePath, branchName, taskID, taskTitle, taskDescription, filesToCommit)
		if !commitResult.Success {
			return commitResult
		}

		return TaskExecutionResult{
			Success:      true,
			Message:      fmt.Sprintf("Successfully executed task %d, committed %d files, and pushed to branch '%s' (based on '%s')", taskID, len(changedFiles), branchName, baseBranch),
			BranchName:   branchName,
			FilesChanged: changedFiles,
			ClaudeOutput: claudeResult.Message,
		}
	}

	// No changes detected
	return TaskExecutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Successfully executed task %d but no file changes were detected in worktree at '%s' on branch '%s' (based on '%s')", taskID, worktreePath, branchName, baseBranch),
		BranchName:   branchName,
		FilesChanged: []string{},
		ClaudeOutput: claudeResult.Message,
	}
}

// generateBranchName creates a Git branch name from task ID and title
func generateBranchName(taskID int, taskTitle string) string {
	// Convert title to lowercase and replace spaces/special chars with hyphens
	title := strings.ToLower(taskTitle)
	title = strings.ReplaceAll(title, " ", "-")
	title = strings.ReplaceAll(title, "_", "-")

	// Remove or replace other special characters
	var cleanTitle strings.Builder
	for _, char := range title {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			cleanTitle.WriteRune(char)
		} else if char == '.' || char == '/' || char == '\\' {
			cleanTitle.WriteRune('-')
		}
	}

	// Limit title length to avoid overly long branch names
	titleStr := cleanTitle.String()
	if len(titleStr) > 40 {
		titleStr = titleStr[:40]
	}

	// Remove trailing hyphens
	titleStr = strings.TrimRight(titleStr, "-")

	return fmt.Sprintf("task-%d-%s", taskID, titleStr)
}

// executeGitWorktreeCommands creates a git worktree and sets up the task branch
func (a *App) executeGitWorktreeCommands(mainRepoPath, worktreePath, baseBranch, branchName string) TaskExecutionResult {
	// First, ensure we clean up any existing worktree that might be using this branch
	listCmd := exec.Command("git", "worktree", "list", "--porcelain")
	listCmd.Dir = mainRepoPath
	listOutput, err := listCmd.Output()
	if err == nil {
		// Parse worktree list to find if our branch is already checked out
		lines := strings.Split(string(listOutput), "\n")
		for i := 0; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "worktree ") && i+2 < len(lines) {
				worktreeDir := strings.TrimPrefix(lines[i], "worktree ")
				if i+2 < len(lines) && strings.HasPrefix(lines[i+2], "branch refs/heads/"+branchName) {
					// Found existing worktree with our branch
					fmt.Printf("Cleaning up existing worktree for branch %s at %s\n", branchName, worktreeDir)
					cleanupCmd := exec.Command("git", "worktree", "remove", "--force", worktreeDir)
					cleanupCmd.Dir = mainRepoPath
					cleanupCmd.Run() // Ignore errors
				}
			}
		}
	}

	// Also cleanup our target directory if it exists
	if _, err := os.Stat(worktreePath); err == nil {
		os.RemoveAll(worktreePath)
	}

	// Delete any existing local branch with the same name
	deleteCmd := exec.Command("git", "branch", "-D", branchName)
	deleteCmd.Dir = mainRepoPath
	deleteCmd.Run() // Ignore errors - branch might not exist

	// Create worktree with new task branch directly from base branch
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	cmd.Dir = mainRepoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create worktree: %v. Output: %s", err, string(output)),
		}
	}

	// Verify the worktree was created successfully
	if _, err := os.Stat(filepath.Join(worktreePath, ".git")); err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Worktree created but .git not found: %v", err),
		}
	}

	// Pull latest changes from the base branch to ensure we're up to date
	pullCmd := exec.Command("git", "pull", "origin", baseBranch)
	pullCmd.Dir = worktreePath
	pullOutput, pullErr := pullCmd.CombinedOutput()
	if pullErr != nil {
		// Don't fail if pull fails - this might happen if there are no changes
		// or if the base branch doesn't exist on remote yet
		fmt.Printf("Warning: Failed to pull latest changes in worktree (this may be normal): %v. Output: %s\n", pullErr, string(pullOutput))
	}

	return TaskExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Successfully created worktree and task branch '%s' from '%s'", branchName, baseBranch),
	}
}

// checkForGitChanges checks if there are any uncommitted changes in the worktree
func (a *App) checkForGitChanges(worktreePath string) (bool, []string) {
	// Run git status --porcelain to check for changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error running git status: %v\n", err)
		return false, nil
	}

	fmt.Printf("Git status output:\n%s\n", string(output))
	fmt.Printf("Git status output (hex): %x\n", output)

	// Parse output to get list of changed files
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var changedFiles []string

	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: "XY filename" where X and Y are status codes
		// More robust parsing: find the first space after the status codes
		if len(line) >= 3 {
			// Find the filename part by looking for the first space and taking everything after it
			// Handle potential extra spaces in status codes
			statusPart := line[:2] // First 2 characters are always status codes
			remaining := line[2:]  // Everything after status codes

			// Trim leading spaces to find the actual filename
			filename := strings.TrimLeft(remaining, " \t")

			// Handle renames which have format "oldname -> newname"
			if strings.Contains(filename, " -> ") {
				// For renames, take the new filename
				parts := strings.Split(filename, " -> ")
				if len(parts) == 2 {
					filename = strings.TrimSpace(parts[1])
				}
			}

			// Remove quotes if the filename is quoted (Git does this for filenames with special chars)
			filename = strings.Trim(filename, "\"")
			filename = strings.TrimSpace(filename)

			if filename != "" && len(statusPart) == 2 {
				fmt.Printf("Parsed file: '%s' from line: '%s' (status: '%s')\n", filename, line, statusPart)
				changedFiles = append(changedFiles, filename)
			}
		}
	}

	fmt.Printf("Found %d changed files: %v\n", len(changedFiles), changedFiles)
	return len(changedFiles) > 0, changedFiles
}

// commitAndPushFromWorktree commits and pushes changes from a git worktree
func (a *App) commitAndPushFromWorktree(worktreePath, branchName string, taskID int, taskTitle, taskDescription string, filesChanged []string) TaskExecutionResult {
	// Add all changed files
	if len(filesChanged) > 0 {
		// Try to add specific files that were reported as changed
		failedFiles := []string{}
		for _, file := range filesChanged {
			cmd := exec.Command("git", "add", file)
			cmd.Dir = worktreePath
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Warning: Failed to add file '%s': %v. Output: %s\n", file, err, string(output))
				failedFiles = append(failedFiles, file)
			}
		}

		// If some files failed to add individually, try adding all changes as fallback
		if len(failedFiles) > 0 {
			fmt.Printf("Some files failed individual add, falling back to 'git add .'\n")
			cmd := exec.Command("git", "add", ".")
			cmd.Dir = worktreePath
			output, err := cmd.CombinedOutput()
			if err != nil {
				return TaskExecutionResult{
					Success: false,
					Message: fmt.Sprintf("Failed to add changes (individual files failed: %v, fallback also failed): %v. Output: %s", failedFiles, err, string(output)),
				}
			}
		}
	} else {
		// Add all changes if no specific files were provided
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = worktreePath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return TaskExecutionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to add all changes: %v. Output: %s", err, string(output)),
			}
		}
	}

	// Create commit with detailed message
	commitMsg := fmt.Sprintf("feat: %s\n\nTask #%d: %s\n\n%s", taskTitle, taskID, taskTitle, taskDescription)
	cmd := exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = worktreePath

	// Set author for the commit
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Claude Code",
		"GIT_AUTHOR_EMAIL=claude@anthropic.com",
		"GIT_COMMITTER_NAME=Claude Code",
		"GIT_COMMITTER_EMAIL=claude@anthropic.com",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to commit changes: %v. Output: %s", err, string(output)),
		}
	}

	// Push the new branch to origin
	cmd = exec.Command("git", "push", "origin", branchName)
	cmd.Dir = worktreePath
	output, err = cmd.CombinedOutput()
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to push branch '%s': %v. Output: %s", branchName, err, string(output)),
		}
	}

	return TaskExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Successfully committed and pushed changes to branch '%s'", branchName),
	}
}

// CleanupTaskWorktree removes a git worktree for a completed task
func (a *App) CleanupTaskWorktree(workspaceName string, taskID int) TaskExecutionResult {
	// Validate input parameters
	if strings.TrimSpace(workspaceName) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	if taskID <= 0 {
		return TaskExecutionResult{
			Success: false,
			Message: "Task ID must be a positive integer",
		}
	}

	// Get workspaces to find the target workspace
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return TaskExecutionResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Calculate worktree path
	worktreePath := filepath.Join(filepath.Dir(targetWorkspace.Path), fmt.Sprintf("task-%d-%s", taskID, workspaceName))

	// Check if worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return TaskExecutionResult{
			Success: true,
			Message: fmt.Sprintf("Worktree for task %d already cleaned up", taskID),
		}
	}

	// Get the branch name before removing worktree
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = worktreePath
	branchOutput, branchErr := branchCmd.Output()
	branchName := ""
	if branchErr == nil {
		branchName = strings.TrimSpace(string(branchOutput))
	}

	// Remove worktree using git command
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = targetWorkspace.Path
	output, err := cmd.CombinedOutput()

	// Always try manual directory removal as well
	if removeErr := os.RemoveAll(worktreePath); removeErr != nil && err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to remove worktree: git error: %v (output: %s), manual removal error: %v", err, string(output), removeErr),
		}
	}

	// Clean up the branch if we got its name and it follows the task pattern
	if branchName != "" && strings.HasPrefix(branchName, fmt.Sprintf("task-%d-", taskID)) {
		deleteBranchCmd := exec.Command("git", "branch", "-D", branchName)
		deleteBranchCmd.Dir = targetWorkspace.Path
		deleteBranchOutput, deleteBranchErr := deleteBranchCmd.CombinedOutput()
		if deleteBranchErr != nil {
			fmt.Printf("Warning: Failed to delete branch '%s': %v. Output: %s\n", branchName, deleteBranchErr, string(deleteBranchOutput))
		} else {
			fmt.Printf("Successfully deleted branch '%s'\n", branchName)
		}
	}

	// Prune any dangling worktree references
	pruneCmd := exec.Command("git", "worktree", "prune")
	pruneCmd.Dir = targetWorkspace.Path
	pruneCmd.Run() // Ignore errors

	return TaskExecutionResult{
		Success: true,
		Message: fmt.Sprintf("Successfully cleaned up worktree for task %d", taskID),
	}
}

// DeleteTaskResult represents the result of deleting a task
type DeleteTaskResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeleteTask removes a task completely, including its worktree if it exists
func (a *App) DeleteTask(workspaceName string, taskID int) DeleteTaskResult {
	// Validate input parameters
	if strings.TrimSpace(workspaceName) == "" {
		return DeleteTaskResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	if taskID <= 0 {
		return DeleteTaskResult{
			Success: false,
			Message: "Task ID must be a positive integer",
		}
	}

	// Get workspaces to find the target workspace
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return DeleteTaskResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return DeleteTaskResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Clean up any worktree for this task if it exists
	cleanupResult := a.CleanupTaskWorktree(workspaceName, taskID)
	if !cleanupResult.Success {
		// Log the cleanup error but don't fail the entire operation
		fmt.Printf("Warning: Failed to cleanup worktree for task %d: %s\n", taskID, cleanupResult.Message)
	}

	return DeleteTaskResult{
		Success: true,
		Message: fmt.Sprintf("Successfully deleted task %d and cleaned up any associated worktree", taskID),
	}
}

// StartTaskConversation starts a new Claude session for a task (simplified - no conversation storage)
func (a *App) StartTaskConversation(workspaceName string, taskID int, taskTitle, taskDescription, baseBranch string) TaskExecutionResult {
	// Validate input parameters
	if strings.TrimSpace(workspaceName) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Workspace name cannot be empty",
		}
	}

	if taskID <= 0 {
		return TaskExecutionResult{
			Success: false,
			Message: "Task ID must be a positive integer",
		}
	}

	if strings.TrimSpace(taskTitle) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Task title cannot be empty",
		}
	}

	if strings.TrimSpace(baseBranch) == "" {
		return TaskExecutionResult{
			Success: false,
			Message: "Base branch cannot be empty",
		}
	}

	// Get workspaces to find the target workspace
	workspacesResult := a.GetWorkspaces()
	if !workspacesResult.Success {
		return TaskExecutionResult{
			Success: false,
			Message: workspacesResult.Message,
		}
	}

	// Find the specified workspace
	var targetWorkspace *Workspace
	for i := range workspacesResult.Workspaces {
		if workspacesResult.Workspaces[i].Name == workspaceName {
			targetWorkspace = &workspacesResult.Workspaces[i]
			break
		}
	}

	if targetWorkspace == nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Workspace '%s' not found", workspaceName),
		}
	}

	// Setup Git worktree
	repo, err := git.PlainOpen(targetWorkspace.Path)
	if err != nil {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to open Git repository: %v", err),
		}
	}

	// Fetch from origin
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch from origin: %v", err),
		}
	}

	// Generate branch name and worktree path
	branchName := generateBranchName(taskID, taskTitle)
	worktreePath := filepath.Join(filepath.Dir(targetWorkspace.Path), fmt.Sprintf("task-%d-%s", taskID, workspaceName))

	// Clean up any existing worktree directory with improved error handling
	if _, err := os.Stat(worktreePath); err == nil {
		// Try to remove using git worktree first
		cleanupCmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
		cleanupCmd.Dir = targetWorkspace.Path
		cleanupCmd.Run() // Ignore errors

		// Ensure directory is gone
		os.RemoveAll(worktreePath)
	}

	// Create worktree
	result := a.executeGitWorktreeCommands(targetWorkspace.Path, worktreePath, baseBranch, branchName)
	if !result.Success {
		return result
	}

	// Initialize Claude client with the worktree path
	claudeClient := claude.NewClaudeClient(worktreePath)

	// Start the Claude session
	claudeResult := claudeClient.ExecuteTask(taskID, taskTitle, taskDescription)
	if !claudeResult.Success {
		return TaskExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to start Claude session: %s", claudeResult.Message),
		}
	}

	// Check for changes and commit if found
	hasChanges, changedFiles := a.checkForGitChanges(worktreePath)
	if hasChanges {
		commitResult := a.commitAndPushFromWorktree(worktreePath, branchName, taskID, taskTitle, taskDescription, changedFiles)
		if !commitResult.Success {
			return commitResult
		}
	}

	return TaskExecutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Started Claude session for task %d on branch '%s'", taskID, branchName),
		BranchName:   branchName,
		ClaudeOutput: claudeResult.Message,
		FilesChanged: changedFiles,
		SessionID:    claudeResult.SessionID,
		WorktreePath: worktreePath,
	}
}

// ContinueClaudeSession continues a Claude session using sessionId and worktree path
func (a *App) ContinueClaudeSession(sessionID, userMessage, worktreePath string) ClaudeSessionResult {
	// Log received session ID for debugging
	fmt.Printf("ContinueClaudeSession called with SessionID: %s, WorktreePath: %s\n", sessionID, worktreePath)

	// Validate input
	if strings.TrimSpace(sessionID) == "" {
		return ClaudeSessionResult{
			Success: false,
			Message: "Session ID cannot be empty",
		}
	}

	if strings.TrimSpace(userMessage) == "" {
		return ClaudeSessionResult{
			Success: false,
			Message: "User message cannot be empty",
		}
	}

	if strings.TrimSpace(worktreePath) == "" {
		return ClaudeSessionResult{
			Success: false,
			Message: "Worktree path cannot be empty",
		}
	}

	// Verify the worktree path exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return ClaudeSessionResult{
			Success: false,
			Message: fmt.Sprintf("Worktree path does not exist: %s", worktreePath),
		}
	}

	// Initialize Claude client with the specific worktree path
	claudeClient := claude.NewClaudeClient(worktreePath)

	// Continue the Claude session
	claudeResult := claudeClient.ContinueConversation(sessionID, userMessage)
	if !claudeResult.Success {
		return ClaudeSessionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to continue Claude session: %s", claudeResult.Message),
		}
	}

	// Check for changes and commit/push if found (similar to StartTaskConversation)
	hasChanges, changedFiles := a.checkForGitChanges(worktreePath)
	if hasChanges {
		// Use files reported by Claude if available, otherwise use detected files
		filesToCommit := claudeResult.FilesChanged
		if len(filesToCommit) == 0 {
			filesToCommit = changedFiles
		}

		// Extract branch information for commit and push

		// Get the branch name from git
		cmd := exec.Command("git", "branch", "--show-current")
		cmd.Dir = worktreePath
		branchOutput, err := cmd.Output()
		branchName := "unknown-branch"
		if err == nil {
			branchName = strings.TrimSpace(string(branchOutput))
		}

		// Create a simple commit message for continued session
		commitMsg := fmt.Sprintf("Update from continued Claude session\n\nUser request: %s\n\nFiles modified:\n", userMessage)
		for _, file := range filesToCommit {
			commitMsg += fmt.Sprintf("- %s\n", file)
		}

		// Commit changes
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = worktreePath
		if err := cmd.Run(); err != nil {
			return ClaudeSessionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to stage changes: %v", err),
			}
		}

		cmd = exec.Command("git", "commit", "-m", commitMsg)
		cmd.Dir = worktreePath
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Claude Code",
			"GIT_AUTHOR_EMAIL=claude@anthropic.com",
			"GIT_COMMITTER_NAME=Claude Code",
			"GIT_COMMITTER_EMAIL=claude@anthropic.com",
		)

		if err := cmd.Run(); err != nil {
			return ClaudeSessionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to commit changes: %v", err),
			}
		}

		// Push changes
		cmd = exec.Command("git", "push", "origin", branchName)
		cmd.Dir = worktreePath
		if err := cmd.Run(); err != nil {
			return ClaudeSessionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to push changes to branch '%s': %v", branchName, err),
			}
		}

		return ClaudeSessionResult{
			Success:      true,
			Message:      fmt.Sprintf("Claude session continued successfully. Committed and pushed %d files to branch '%s'", len(filesToCommit), branchName),
			Response:     claudeResult.Message,
			FilesChanged: filesToCommit,
		}
	}

	// No changes detected
	return ClaudeSessionResult{
		Success:      true,
		Message:      "Claude session continued successfully (no file changes detected)",
		Response:     claudeResult.Message,
		FilesChanged: []string{},
	}
}
