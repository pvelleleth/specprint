import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Epic, Task } from './types';

interface EpicCardProps {
  epic: Epic;
  tasks: Task[];
  onEpicClick: (epicId: number) => void;
  isSelected: boolean;
}

export function EpicCard({ epic, tasks, onEpicClick, isSelected }: EpicCardProps) {
  const epicTasks = tasks.filter(task => 
    task.title.includes(`[${epic.title}]`) || 
    task.description.includes(`Epic: ${epic.title}`)
  );
  
  const completedTasks = epicTasks.filter(task => task.status === 'done');
  const inProgressTasks = epicTasks.filter(task => task.status === 'in-progress');
  const todoTasks = epicTasks.filter(task => task.status === 'todo');
  
  const progressPercentage = epicTasks.length > 0 
    ? Math.round((completedTasks.length / epicTasks.length) * 100)
    : 0;

  const getProgressColor = (percentage: number) => {
    if (percentage >= 80) return 'bg-green-500';
    if (percentage >= 50) return 'bg-yellow-500';
    if (percentage >= 25) return 'bg-orange-500';
    return 'bg-red-500';
  };

  return (
    <Card 
      className={`cursor-pointer transition-all duration-200 hover:shadow-md ${
        isSelected ? 'ring-2 ring-blue-500 bg-blue-50' : ''
      }`}
      onClick={() => onEpicClick(epic.id)}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <CardTitle className="text-sm font-semibold text-gray-900">
            {epic.title}
          </CardTitle>
          <div className="flex items-center gap-2">
            <span className="px-2 py-1 text-xs rounded-full bg-blue-100 text-blue-700 font-medium">
              Epic #{epic.id}
            </span>
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0">
        <p className="text-xs text-gray-600 mb-3 line-clamp-2">
          {epic.description}
        </p>
        
        {/* Progress Bar */}
        <div className="mb-3">
          <div className="flex items-center justify-between text-xs text-gray-600 mb-1">
            <span>Progress</span>
            <span>{progressPercentage}%</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div 
              className={`h-2 rounded-full transition-all duration-300 ${getProgressColor(progressPercentage)}`}
              style={{ width: `${progressPercentage}%` }}
            ></div>
          </div>
        </div>

        {/* Task Counts */}
        <div className="flex items-center justify-between text-xs">
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1">
              <div className="w-2 h-2 bg-gray-400 rounded-full"></div>
              {todoTasks.length} todo
            </span>
            <span className="flex items-center gap-1">
              <div className="w-2 h-2 bg-yellow-400 rounded-full"></div>
              {inProgressTasks.length} active
            </span>
            <span className="flex items-center gap-1">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              {completedTasks.length} done
            </span>
          </div>
          <span className="text-gray-500">
            {epic.stories.length} stories
          </span>
        </div>
      </CardContent>
    </Card>
  );
} 