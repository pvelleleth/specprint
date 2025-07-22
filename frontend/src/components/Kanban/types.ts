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
}

export interface BoardState {
  tasks: Task[];
  lastUpdated: string;
} 