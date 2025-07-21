import { main } from "../../../wailsjs/go/models";

// Extend the generated Task type to include status for our Kanban board
export interface TaskWithStatus extends main.Task {
  status: string;
}

// Type alias for convenience
export type Task = TaskWithStatus;

// Re-export the generated types for consistency
export type Epic = main.Epic;
export type Story = main.Story;
export type Workspace = main.Workspace;

export interface BoardState {
  tasks: Task[];
  epics: Epic[];
  stories: Story[];
  lastUpdated: string;
}

export type ViewMode = 'flat' | 'hierarchical'; 