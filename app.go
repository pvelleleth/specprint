package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
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
			for _, repo := range repos {
				if repo.IsDir() {
					repoPath := filepath.Join(baseDir, repo.Name())

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
		HasPRD:     false,
	}

	// Get existing workspaces and add this one
	workspacesResult := a.GetWorkspaces()
	workspaces := workspacesResult.Workspaces
	workspaces = append(workspaces, workspace)
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
