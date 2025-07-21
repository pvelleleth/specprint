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

// Epic represents a high-level feature or major component
type Epic struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Stories     []Story `json:"stories"`
}

// Story represents a user story or functional requirement
type Story struct {
	ID          int    `json:"id"`
	EpicID      int    `json:"epicId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tasks       []Task `json:"tasks"`
}

// Task represents a single implementation task
type Task struct {
	ID           int    `json:"id"`
	StoryID      int    `json:"storyId"`
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
	Epics   []Epic `json:"epics,omitempty"`
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
	systemPrompt := `You are an expert project manager and software architect. Your task is to analyze a Product Requirements Document (PRD) and generate a hierarchical structure of Epics > Stories > Tasks.

STRUCTURE:
- EPICS: High-level features or major components (3-6 epics)
- STORIES: User stories or functional requirements within each epic (2-5 stories per epic)
- TASKS: Specific implementation tasks within each story (3-8 tasks per story)

For each level, provide:

EPIC:
- id: Unique sequential number starting from 1
- title: High-level feature name (max 60 characters)
- description: Brief overview of the epic (max 150 characters)
- stories: Array of stories within this epic

STORY:
- id: Unique sequential number starting from 1
- epicId: ID of the parent epic
- title: User story title (max 80 characters)
- description: User story description (max 200 characters)
- tasks: Array of tasks within this story

TASK:
- id: Unique sequential number starting from 1
- storyId: ID of the parent story
- title: Specific task title (max 80 characters)
- description: What needs to be done (max 200 characters)
- dependencies: Array of task IDs that must be completed first (use [] if none)
- priority: "high", "medium", or "low"
- estimate: Time estimate like "2h", "1d", "3d"

RULES:
1. Organize by major functional areas (Epics)
2. Break down into user-focused stories
3. Create specific, actionable tasks
4. Consider dependencies and sequencing
5. Include setup, implementation, testing phases
6. Provide realistic time estimates
7. Aim for 3-6 epics, 2-5 stories per epic, 3-8 tasks per story

Return ONLY a valid JSON array of epics. Do not include any other text or formatting.

Example format:
[
  {
    "id": 1,
    "title": "User Authentication",
    "description": "Complete user registration and login system",
    "stories": [
      {
        "id": 1,
        "epicId": 1,
        "title": "User Registration",
        "description": "Allow new users to create accounts",
        "tasks": [
          {
            "id": 1,
            "storyId": 1,
            "title": "Registration Form UI",
            "description": "Create user registration form with validation",
            "dependencies": [],
            "priority": "high",
            "estimate": "4h"
          }
        ]
      }
    ]
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
	var epics []Epic
	err = json.Unmarshal([]byte(responseContent), &epics)
	if err != nil {
		return TaskGenerationResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse JSON response: %v. Response was: %s", err, responseContent),
		}
	}

	// Validate the parsed epics
	if len(epics) == 0 {
		return TaskGenerationResult{
			Success: false,
			Message: "No epics were generated from the PRD",
		}
	}

	// Validate epic structure and count total tasks
	totalTasks := 0
	for i, epic := range epics {
		if epic.ID <= 0 {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Epic %d has invalid ID: %d", i+1, epic.ID),
			}
		}
		if strings.TrimSpace(epic.Title) == "" {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Epic %d has empty title", epic.ID),
			}
		}
		if strings.TrimSpace(epic.Description) == "" {
			return TaskGenerationResult{
				Success: false,
				Message: fmt.Sprintf("Epic %d has empty description", epic.ID),
			}
		}

		// Validate stories within epic
		for j, story := range epic.Stories {
			if story.ID <= 0 {
				return TaskGenerationResult{
					Success: false,
					Message: fmt.Sprintf("Story %d in Epic %d has invalid ID: %d", j+1, epic.ID, story.ID),
				}
			}
			if story.EpicID != epic.ID {
				return TaskGenerationResult{
					Success: false,
					Message: fmt.Sprintf("Story %d has incorrect epicId: expected %d, got %d", story.ID, epic.ID, story.EpicID),
				}
			}
			if strings.TrimSpace(story.Title) == "" {
				return TaskGenerationResult{
					Success: false,
					Message: fmt.Sprintf("Story %d in Epic %d has empty title", story.ID, epic.ID),
				}
			}
			if strings.TrimSpace(story.Description) == "" {
				return TaskGenerationResult{
					Success: false,
					Message: fmt.Sprintf("Story %d in Epic %d has empty description", story.ID, epic.ID),
				}
			}

			// Validate tasks within story
			for k, task := range story.Tasks {
				if task.ID <= 0 {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d in Story %d has invalid ID: %d", k+1, story.ID, task.ID),
					}
				}
				if task.StoryID != story.ID {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d has incorrect storyId: expected %d, got %d", task.ID, story.ID, task.StoryID),
					}
				}
				if strings.TrimSpace(task.Title) == "" {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d in Story %d has empty title", task.ID, story.ID),
					}
				}
				if strings.TrimSpace(task.Description) == "" {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d in Story %d has empty description", task.ID, story.ID),
					}
				}
				if strings.TrimSpace(task.Priority) == "" {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d in Story %d has empty priority", task.ID, story.ID),
					}
				}
				if strings.TrimSpace(task.Estimate) == "" {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d in Story %d has empty estimate", task.ID, story.ID),
					}
				}
				if task.Dependencies == nil {
					return TaskGenerationResult{
						Success: false,
						Message: fmt.Sprintf("Task %d in Story %d has nil dependencies", task.ID, story.ID),
					}
				}
				totalTasks++
			}
		}
	}

	return TaskGenerationResult{
		Success: true,
		Message: fmt.Sprintf("Successfully generated %d epics with %d total tasks from PRD", len(epics), totalTasks),
		Epics:   epics,
	}
}

// FlattenTasks converts the hierarchical Epic > Story > Task structure into a flat list of tasks
// This is useful for backward compatibility and simpler processing
func (a *App) FlattenTasks(epics []Epic) []Task {
	var flatTasks []Task

	for _, epic := range epics {
		for _, story := range epic.Stories {
			for _, task := range story.Tasks {
				// Create a flattened task with epic and story context
				flatTask := Task{
					ID:           task.ID,
					StoryID:      task.StoryID,
					Title:        fmt.Sprintf("[%s] %s", epic.Title, task.Title),
					Description:  fmt.Sprintf("Epic: %s | Story: %s | %s", epic.Title, story.Title, task.Description),
					Dependencies: task.Dependencies,
					Priority:     task.Priority,
					Estimate:     task.Estimate,
				}
				flatTasks = append(flatTasks, flatTask)
			}
		}
	}

	return flatTasks
}

// GetTaskSummary provides a summary of the generated tasks
func (a *App) GetTaskSummary(epics []Epic) map[string]interface{} {
	totalTasks := 0
	totalStories := 0
	priorityCounts := map[string]int{"high": 0, "medium": 0, "low": 0}

	for _, epic := range epics {
		totalStories += len(epic.Stories)
		for _, story := range epic.Stories {
			totalTasks += len(story.Tasks)
			for _, task := range story.Tasks {
				priorityCounts[task.Priority]++
			}
		}
	}

	return map[string]interface{}{
		"totalEpics":     len(epics),
		"totalStories":   totalStories,
		"totalTasks":     totalTasks,
		"priorityCounts": priorityCounts,
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
	return a.GenerateTasks(string(prdContent))
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
