package main

import (
	"strings"
	"testing"
)

func TestGenerateTasks(t *testing.T) {
	app := NewApp()

	// Test case 1: Valid PRD content
	samplePRD := `# Product Requirements Document: Simple Task Manager

## Overview
Build a basic task management web application.

## Features
1. User registration and authentication
2. Create, read, update, delete tasks
3. Task prioritization (High, Medium, Low)
4. Task status tracking (Todo, In Progress, Done)
5. Simple dashboard with task overview

## Technical Requirements
- Frontend: React with TypeScript
- Backend: Node.js with Express
- Database: PostgreSQL
- Authentication: JWT tokens
- API: RESTful endpoints

## User Stories
- As a user, I want to register and login
- As a user, I want to create new tasks
- As a user, I want to mark tasks as complete
- As a user, I want to see my task dashboard`

	result := app.GenerateTasks(samplePRD)

	// Verify success
	if !result.Success {
		t.Errorf("Expected success, got error: %s", result.Message)
		return
	}

	// Verify epics were generated
	if len(result.Epics) == 0 {
		t.Error("Expected epics to be generated, got empty array")
		return
	}

	// Verify epic structure and count total tasks
	totalTasks := 0
	for i, epic := range result.Epics {
		if epic.ID <= 0 {
			t.Errorf("Epic %d has invalid ID: %d", i, epic.ID)
		}
		if strings.TrimSpace(epic.Title) == "" {
			t.Errorf("Epic %d has empty title", epic.ID)
		}
		if strings.TrimSpace(epic.Description) == "" {
			t.Errorf("Epic %d has empty description", epic.ID)
		}

		// Verify stories within epic
		for j, story := range epic.Stories {
			if story.ID <= 0 {
				t.Errorf("Story %d in Epic %d has invalid ID: %d", j, epic.ID, story.ID)
			}
			if story.EpicID != epic.ID {
				t.Errorf("Story %d has incorrect epicId: expected %d, got %d", story.ID, epic.ID, story.EpicID)
			}
			if strings.TrimSpace(story.Title) == "" {
				t.Errorf("Story %d in Epic %d has empty title", story.ID, epic.ID)
			}
			if strings.TrimSpace(story.Description) == "" {
				t.Errorf("Story %d in Epic %d has empty description", story.ID, epic.ID)
			}

			// Verify tasks within story
			for k, task := range story.Tasks {
				if task.ID <= 0 {
					t.Errorf("Task %d in Story %d has invalid ID: %d", k, story.ID, task.ID)
				}
				if task.StoryID != story.ID {
					t.Errorf("Task %d has incorrect storyId: expected %d, got %d", task.ID, story.ID, task.StoryID)
				}
				if strings.TrimSpace(task.Title) == "" {
					t.Errorf("Task %d in Story %d has empty title", task.ID, story.ID)
				}
				if strings.TrimSpace(task.Description) == "" {
					t.Errorf("Task %d in Story %d has empty description", task.ID, story.ID)
				}
				if strings.TrimSpace(task.Priority) == "" {
					t.Errorf("Task %d in Story %d has empty priority", task.ID, story.ID)
				}
				if strings.TrimSpace(task.Estimate) == "" {
					t.Errorf("Task %d in Story %d has empty estimate", task.ID, story.ID)
				}
				if task.Dependencies == nil {
					t.Errorf("Task %d in Story %d has nil dependencies", task.ID, story.ID)
				}
				totalTasks++
			}
		}
	}

	t.Logf("✅ Successfully generated %d epics with %d total tasks", len(result.Epics), totalTasks)
	for _, epic := range result.Epics {
		t.Logf("Epic %d: %s", epic.ID, epic.Title)
		for _, story := range epic.Stories {
			t.Logf("  Story %d: %s", story.ID, story.Title)
			for _, task := range story.Tasks {
				t.Logf("    Task %d: %s (%s priority, %s estimate)", task.ID, task.Title, task.Priority, task.Estimate)
			}
		}
	}
}

func TestGenerateTasksEmptyContent(t *testing.T) {
	app := NewApp()

	// Test case 2: Empty PRD content
	result := app.GenerateTasks("")

	if result.Success {
		t.Error("Expected failure for empty content, got success")
	}

	if !strings.Contains(result.Message, "empty") {
		t.Errorf("Expected error message about empty content, got: %s", result.Message)
	}
}

func TestGenerateTasksWhitespaceContent(t *testing.T) {
	app := NewApp()

	// Test case 3: Whitespace-only PRD content
	result := app.GenerateTasks("   \n\t   ")

	if result.Success {
		t.Error("Expected failure for whitespace-only content, got success")
	}

	if !strings.Contains(result.Message, "empty") {
		t.Errorf("Expected error message about empty content, got: %s", result.Message)
	}
}

func TestGenerateTasksFromWorkspacePRD(t *testing.T) {
	app := NewApp()

	// Test case 4: Non-existent workspace
	result := app.GenerateTasksFromWorkspacePRD("non-existent-workspace")

	if result.Success {
		t.Error("Expected failure for non-existent workspace, got success")
	}

	if !strings.Contains(result.Message, "not found") {
		t.Errorf("Expected error message about workspace not found, got: %s", result.Message)
	}
}

func TestGenerateTasksFromWorkspacePRDEmptyName(t *testing.T) {
	app := NewApp()

	// Test case 5: Empty workspace name
	result := app.GenerateTasksFromWorkspacePRD("")

	if result.Success {
		t.Error("Expected failure for empty workspace name, got success")
	}

	if !strings.Contains(result.Message, "empty") {
		t.Errorf("Expected error message about empty workspace name, got: %s", result.Message)
	}
}

func TestFlattenTasks(t *testing.T) {
	app := NewApp()

	// Create sample epics structure
	epics := []Epic{
		{
			ID:          1,
			Title:       "User Auth",
			Description: "Authentication system",
			Stories: []Story{
				{
					ID:          1,
					EpicID:      1,
					Title:       "Login",
					Description: "User login functionality",
					Tasks: []Task{
						{
							ID:           1,
							StoryID:      1,
							Title:        "Login Form",
							Description:  "Create login form",
							Dependencies: []int{},
							Priority:     "high",
							Estimate:     "4h",
						},
					},
				},
			},
		},
	}

	flatTasks := app.FlattenTasks(epics)

	if len(flatTasks) != 1 {
		t.Errorf("Expected 1 flattened task, got %d", len(flatTasks))
	}

	expectedTitle := "[User Auth] Login Form"
	if flatTasks[0].Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, flatTasks[0].Title)
	}

	expectedDesc := "Epic: User Auth | Story: Login | Create login form"
	if flatTasks[0].Description != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, flatTasks[0].Description)
	}
}

func TestGetTaskSummary(t *testing.T) {
	app := NewApp()

	// Create sample epics structure
	epics := []Epic{
		{
			ID:          1,
			Title:       "User Auth",
			Description: "Authentication system",
			Stories: []Story{
				{
					ID:          1,
					EpicID:      1,
					Title:       "Login",
					Description: "User login functionality",
					Tasks: []Task{
						{
							ID:           1,
							StoryID:      1,
							Title:        "Login Form",
							Description:  "Create login form",
							Dependencies: []int{},
							Priority:     "high",
							Estimate:     "4h",
						},
						{
							ID:           2,
							StoryID:      1,
							Title:        "API Integration",
							Description:  "Connect to auth API",
							Dependencies: []int{1},
							Priority:     "medium",
							Estimate:     "2d",
						},
					},
				},
			},
		},
	}

	summary := app.GetTaskSummary(epics)

	if summary["totalEpics"] != 1 {
		t.Errorf("Expected 1 epic, got %v", summary["totalEpics"])
	}

	if summary["totalStories"] != 1 {
		t.Errorf("Expected 1 story, got %v", summary["totalStories"])
	}

	if summary["totalTasks"] != 2 {
		t.Errorf("Expected 2 tasks, got %v", summary["totalTasks"])
	}

	priorityCounts := summary["priorityCounts"].(map[string]int)
	if priorityCounts["high"] != 1 {
		t.Errorf("Expected 1 high priority task, got %d", priorityCounts["high"])
	}

	if priorityCounts["medium"] != 1 {
		t.Errorf("Expected 1 medium priority task, got %d", priorityCounts["medium"])
	}
}
