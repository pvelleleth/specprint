import React from 'react';
import { useDrag } from 'react-dnd';
import { Task } from './types';

interface TaskCardProps {
  task: Task;
  columnId: string;
  allTasks?: Task[]; // Add allTasks prop to check dependency status
}

const ItemType = 'TASK';

export function TaskCard({ task, columnId, allTasks = [] }: TaskCardProps) {
  const [{ isDragging }, drag] = useDrag({
    type: ItemType,
    item: { task, sourceColumnId: columnId },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

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
      className={`p-4 bg-white rounded-lg shadow-sm border-l-4 ${getBorderColor(task.status)} ${isDragging ? 'opacity-50' : ''} ${hasUncompletedDependencies ? 'opacity-60' : ''}`}
    >
      <div className="flex justify-between items-start">
        <h4 className="font-semibold text-base text-gray-800 leading-tight">{task.title}</h4>
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-gray-500 bg-gray-100 px-2 py-1 rounded">
            #{task.id}
          </span>
          <span className={`text-xs px-2 py-1 rounded font-medium ${getPriorityColor(task.priority)}`}>
            {task.priority}
          </span>
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
  );
} 