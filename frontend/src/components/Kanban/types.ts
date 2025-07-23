// Remove import
// import { main } from "../../../wailsjs/go/models";

// Remove old TaskWithStatus and aliases
export interface Task {
  id: number;
  title: string;
  description: string;
  dependencies: number[];
  priority: string;
  estimate: string;
  status: string;
  isRunning?: boolean; // Add running state for individual tasks
  worktreePath?: string; // Add worktree path for tracking active worktrees
  sessionId?: string;
}

export interface BoardState {
  tasks: Task[];
  lastUpdated: string;
} 