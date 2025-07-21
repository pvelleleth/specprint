import React from 'react';
import { useDrop } from 'react-dnd';
import { TaskCard } from './TaskCard';
import { Story, Task } from './types';

interface StoryLaneProps {
  story: Story;
  tasks: Task[];
  columnId: string;
  onTaskMove: (task: Task, sourceColumnId: string, targetColumnId: string) => void;
  isCollapsed: boolean;
  onToggleCollapse: (storyId: number) => void;
}

const ItemType = 'TASK';

export function StoryLane({ 
  story, 
  tasks, 
  columnId, 
  onTaskMove, 
  isCollapsed, 
  onToggleCollapse 
}: StoryLaneProps) {
  const [{ isOver }, drop] = useDrop({
    accept: ItemType,
    drop: (item: { task: Task; sourceColumnId: string }) => {
      if (item.sourceColumnId !== columnId) {
        onTaskMove(item.task, item.sourceColumnId, columnId);
      }
    },
    collect: (monitor) => ({
      isOver: monitor.isOver(),
    }),
  });

  const completedTasks = tasks.filter(task => task.status === 'done');
  const progressPercentage = tasks.length > 0 
    ? Math.round((completedTasks.length / tasks.length) * 100)
    : 0;

  const getProgressColor = (percentage: number) => {
    if (percentage >= 80) return 'bg-green-400';
    if (percentage >= 50) return 'bg-yellow-400';
    if (percentage >= 25) return 'bg-orange-400';
    return 'bg-red-400';
  };

  return (
    <div className="mb-4 border rounded-lg bg-white shadow-sm">
      {/* Story Header */}
      <div 
        className="p-3 bg-gray-50 border-b cursor-pointer hover:bg-gray-100 transition-colors"
        onClick={() => onToggleCollapse(story.id)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <svg 
              className={`w-4 h-4 text-gray-500 transition-transform ${isCollapsed ? '' : 'rotate-90'}`}
              fill="none" 
              stroke="currentColor" 
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            <h4 className="font-medium text-sm text-gray-900">{story.title}</h4>
            <span className="px-2 py-1 text-xs rounded-full bg-purple-100 text-purple-700 font-medium">
              Story #{story.id}
            </span>
          </div>
          
          <div className="flex items-center gap-3">
            {/* Progress indicator */}
            <div className="flex items-center gap-2">
              <div className="w-16 bg-gray-200 rounded-full h-1.5">
                <div 
                  className={`h-1.5 rounded-full transition-all duration-300 ${getProgressColor(progressPercentage)}`}
                  style={{ width: `${progressPercentage}%` }}
                ></div>
              </div>
              <span className="text-xs text-gray-600">{tasks.length}</span>
            </div>
          </div>
        </div>
        
        {!isCollapsed && (
          <p className="text-xs text-gray-600 mt-2 line-clamp-2">
            {story.description}
          </p>
        )}
      </div>

      {/* Story Tasks */}
      {!isCollapsed && (
        <div
          ref={drop}
          className={`p-3 min-h-20 ${isOver ? 'bg-blue-50 border-blue-200' : ''} transition-all duration-200`}
        >
          {tasks.length === 0 ? (
            <div className="text-center py-4 text-gray-400">
              <p className="text-xs">No tasks in this story</p>
              {isOver && (
                <p className="text-xs mt-1 text-blue-600">Drop task here</p>
              )}
            </div>
          ) : (
            <div className="space-y-0">
              {tasks.map((task) => (
                <TaskCard
                  key={task.id}
                  task={task}
                  columnId={columnId}
                />
              ))}
              {isOver && (
                <div className="h-2 bg-blue-200 rounded-full opacity-50 animate-pulse"></div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
} 