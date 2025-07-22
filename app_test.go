package main

import (
	"fmt"
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

	// Verify tasks were generated
	if len(result.Tasks) == 0 {
		t.Error("Expected tasks to be generated, got empty array")
		return
	}

	// Verify task structure
	for i, task := range result.Tasks {
		if task.ID <= 0 {
			t.Errorf("Task %d has invalid ID: %d", i, task.ID)
		}
		if strings.TrimSpace(task.Title) == "" {
			t.Errorf("Task %d has empty title", task.ID)
		}
		if strings.TrimSpace(task.Description) == "" {
			t.Errorf("Task %d has empty description", task.ID)
		}
		if strings.TrimSpace(task.Priority) == "" {
			t.Errorf("Task %d has empty priority", task.ID)
		}
		if strings.TrimSpace(task.Estimate) == "" {
			t.Errorf("Task %d has empty estimate", task.ID)
		}
		if task.Dependencies == nil {
			t.Errorf("Task %d has nil dependencies", task.ID)
		}

		// Verify priority is valid
		validPriorities := []string{"high", "medium", "low"}
		priorityValid := false
		for _, validPriority := range validPriorities {
			if strings.ToLower(task.Priority) == validPriority {
				priorityValid = true
				break
			}
		}
		if !priorityValid {
			t.Errorf("Task %d has invalid priority: %s", task.ID, task.Priority)
		}

		// Verify dependencies reference valid task IDs
		for _, depID := range task.Dependencies {
			found := false
			for _, otherTask := range result.Tasks {
				if otherTask.ID == depID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Task %d references non-existent dependency: %d", task.ID, depID)
			}
		}
	}

	t.Logf("✅ Successfully generated %d tasks", len(result.Tasks))
	for _, task := range result.Tasks {
		deps := "none"
		if len(task.Dependencies) > 0 {
			deps = fmt.Sprintf("%v", task.Dependencies)
		}
		t.Logf("Task %d: %s (%s priority, %s estimate, deps: %s)",
			task.ID, task.Title, task.Priority, task.Estimate, deps)
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
