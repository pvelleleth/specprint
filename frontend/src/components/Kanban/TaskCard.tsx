import React from 'react';
import { useDrag } from 'react-dnd';
import { Task } from './types';

interface TaskCardProps {
  task: Task;
  columnId: string;
  allTasks?: Task[]; // Add allTasks prop to check dependency status
  onEditTask?: (task: Task) => void; // Add callback for editing
  onRunTask?: (task: Task, baseBranch: string) => void; // Updated signature
}

const ItemType = 'TASK';

export function TaskCard({ task, columnId, allTasks = [], onEditTask, onRunTask }: TaskCardProps) {
  const [{ isDragging }, drag] = useDrag({
    type: ItemType,
    item: { task, sourceColumnId: columnId },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

  const handleCardClick = (e: React.MouseEvent) => {
    // Prevent drag when clicking edit button or card
    e.stopPropagation();
    if (onEditTask) {
      onEditTask(task);
    }
  };

  const getBorderColor = (status: string) => {
    switch (status) {
      case 'todo':
        return 'border-blue-400';
      case 'in-progress':
        return 'border-yellow-400';
      case 'done':
        return 'border-green-400';
      default:
        return 'border-gray-300';
    }
  };

  // Check if task has dependencies and if they're completed
  const hasDependencies = task.dependencies && task.dependencies.length > 0;
  const dependencyTasks = allTasks.filter(t => task.dependencies.includes(t.id));
  const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
  const allDependenciesCompleted = hasDependencies && completedDependencies.length === task.dependencies.length;
  const hasUncompletedDependencies = hasDependencies && !allDependenciesCompleted;

  // Get priority color
  const getPriorityColor = (priority: string) => {
    switch (priority.toLowerCase()) {
      case 'high':
        return 'bg-red-100 text-red-800';
      case 'medium':
        return 'bg-yellow-100 text-yellow-800';
      case 'low':
        return 'bg-green-100 text-green-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div
      ref={drag}
      className={`p-4 bg-white rounded-lg shadow-sm border-l-4 ${getBorderColor(task.status)} ${isDragging ? 'opacity-50' : ''} ${hasUncompletedDependencies ? 'opacity-60' : ''} cursor-pointer hover:shadow-md transition-shadow duration-200`}
      onClick={handleCardClick}
    >
      {/* Progress bar for running tasks */}
      {task.isRunning && (
        <div className="mb-3 -mx-4 -mt-4 px-4 pt-2">
          <div className="flex items-center gap-2 mb-2">
            <svg className="w-4 h-4 text-blue-600 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            <span className="text-sm text-blue-600 font-medium">Task is running...</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div className="bg-blue-600 h-2 rounded-full animate-pulse" style={{ width: '100%' }}></div>
          </div>
        </div>
      )}

      <div className="flex justify-between items-start">
        <h4 className="font-semibold text-base text-gray-800 leading-tight">{task.title}</h4>
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-gray-500 bg-gray-100 px-2 py-1 rounded">
            #{task.id}
          </span>
          <span className={`text-xs px-2 py-1 rounded font-medium ${getPriorityColor(task.priority)}`}>
            {task.priority}
          </span>
          {/* Edit icon */}
          <button
            onClick={(e) => {
              e.stopPropagation();
              if (onEditTask) onEditTask(task);
            }}
            className="p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors duration-200"
            title="Edit task"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
        </div>
      </div>
      
      <p className="mt-2 text-sm text-gray-600">
        {task.description}
      </p>

      {/* Dependencies Section */}
      {hasDependencies && (
        <div className="mt-3 pt-2 border-t border-gray-100">
          <div className="flex items-center gap-2 mb-1">
            <svg className="w-3 h-3 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
            <span className="text-xs font-medium text-gray-600">
              Dependencies ({completedDependencies.length}/{task.dependencies.length})
            </span>
          </div>
          
          <div className="flex flex-wrap gap-1">
            {task.dependencies.map(depId => {
              const depTask = allTasks.find(t => t.id === depId);
              const isCompleted = depTask?.status === 'done';
              return (
                <span
                  key={depId}
                  className={`text-xs px-2 py-1 rounded font-mono ${
                    isCompleted 
                      ? 'bg-green-100 text-green-700 border border-green-200' 
                      : 'bg-red-100 text-red-700 border border-red-200'
                  }`}
                  title={depTask ? `${depTask.title} (${depTask.status})` : `Task #${depId}`}
                >
                  #{depId}
                </span>
              );
            })}
          </div>
          
          {hasUncompletedDependencies && (
            <div className="mt-1 text-xs text-amber-600 flex items-center gap-1">
              <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
              </svg>
              Complete dependencies first
            </div>
          )}
        </div>
      )}

      {/* Estimate */}
      <div className="mt-2 flex items-center justify-between">
        <span className="text-xs text-gray-500">
          Estimate: {task.estimate}
        </span>
        <div className="flex items-center gap-2">
          {task.worktreePath && (
            <span className="text-xs px-2 py-1 rounded bg-purple-100 text-purple-700 border border-purple-200" title="Has active worktree">
              🔧 Worktree
            </span>
          )}
          {hasDependencies && (
            <span className={`text-xs px-2 py-1 rounded ${
              allDependenciesCompleted 
                ? 'bg-green-100 text-green-700' 
                : 'bg-amber-100 text-amber-700'
            }`}>
              {allDependenciesCompleted ? 'Ready' : 'Blocked'}
            </span>
          )}
        </div>
      </div>
    </div>
  );
} 