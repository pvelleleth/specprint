import React from 'react';
import { useDrag } from 'react-dnd';
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Task } from './types';

interface TaskCardProps {
  task: Task;
  columnId: string;
}

const ItemType = 'TASK';

export function TaskCard({ task, columnId }: TaskCardProps) {
  const [{ isDragging }, drag] = useDrag({
    type: ItemType,
    item: { task, sourceColumnId: columnId },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

  const getPriorityColor = (priority: string) => {
    switch (priority.toLowerCase()) {
      case 'high':
        return 'border-l-red-500 bg-red-50';
      case 'medium':
        return 'border-l-yellow-500 bg-yellow-50';
      case 'low':
        return 'border-l-green-500 bg-green-50';
      default:
        return 'border-l-gray-500 bg-gray-50';
    }
  };

  const getDependencyStatus = (dependencies: number[], allTasks: Task[]) => {
    if (dependencies.length === 0) return null;
    
    const completedDeps = dependencies.filter(depId => 
      allTasks.find(t => t.id === depId)?.status === 'done'
    ).length;
    
    return `${completedDeps}/${dependencies.length} deps completed`;
  };

  return (
    <div
      ref={drag}
      className={`cursor-move transition-opacity ${isDragging ? 'opacity-50' : 'opacity-100'}`}
    >
      <Card className={`mb-3 border-l-4 ${getPriorityColor(task.priority)} hover:shadow-md transition-shadow`}>
        <CardHeader className="pb-3">
          <div className="flex items-start justify-between">
            <CardTitle className="text-sm font-medium leading-5 text-gray-900">
              {task.title}
            </CardTitle>
            <div className="flex items-center gap-1">
              <span className={`px-2 py-1 text-xs rounded-full font-medium ${
                task.priority === 'high' ? 'bg-red-100 text-red-700' :
                task.priority === 'medium' ? 'bg-yellow-100 text-yellow-700' :
                'bg-green-100 text-green-700'
              }`}>
                {task.priority}
              </span>
              <span className="px-2 py-1 text-xs rounded-full bg-blue-100 text-blue-700 font-medium">
                {task.estimate}
              </span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <p className="text-sm text-gray-600 mb-3 leading-5">
            {task.description}
          </p>
          
          <div className="flex items-center justify-between text-xs text-gray-500">
            <span>Task #{task.id}</span>
            {task.dependencies.length > 0 && (
              <span className="flex items-center gap-1">
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.102m0-3.464L7.464 8.5a4 4 0 005.656-5.656L14.828 1.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656L13.828 10.172z" />
                </svg>
                {task.dependencies.length} deps
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
} 