import React from 'react';
import { useDrop } from 'react-dnd';
import { TaskCard } from './TaskCard';
import { Task } from './types';

interface EnhancedKanbanColumnProps {
  title: string;
  columnId: string;
  tasks: Task[];
  allTasks: Task[]; // Add allTasks prop
  onTaskMove: (task: Task, sourceColumnId: string, targetColumnId: string) => void;
}

const ItemType = 'TASK';

export function EnhancedKanbanColumn({ 
  title, 
  columnId, 
  tasks, 
  allTasks,
  onTaskMove
}: EnhancedKanbanColumnProps) {
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

  const getColumnColor = (columnId: string) => {
    switch (columnId) {
      case 'todo':
        return 'bg-blue-50 border-blue-200';
      case 'in-progress':
        return 'bg-yellow-50 border-yellow-200';
      case 'done':
        return 'bg-green-50 border-green-200';
      default:
        return 'bg-gray-50 border-gray-200';
    }
  };

  const getColumnHeaderColor = (columnId: string) => {
    switch (columnId) {
      case 'todo':
        return 'bg-blue-100 text-blue-800';
      case 'in-progress':
        return 'bg-yellow-100 text-yellow-800';
      case 'done':
        return 'bg-green-100 text-green-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const renderFlatView = () => (
    <div className="space-y-3">
      {tasks.map((task) => (
        <TaskCard
          key={task.id}
          task={task}
          columnId={columnId}
          allTasks={allTasks}
        />
      ))}
      {isOver && (
        <div className="h-2 bg-blue-200 rounded-full opacity-50 animate-pulse"></div>
      )}
    </div>
  );

  return (
    <div className="flex-1 min-w-80">
      <div className={`rounded-lg border-2 ${getColumnColor(columnId)} ${isOver ? 'border-dashed border-opacity-75' : ''} h-full`}>
        {/* Column Header */}
        <div className={`p-4 rounded-t-lg ${getColumnHeaderColor(columnId)}`}>
          <div className="flex items-center justify-between">
            <h3 className="font-semibold text-lg">{title}</h3>
            <span className="px-2 py-1 text-sm rounded-full bg-white bg-opacity-50 font-medium">
              {tasks.length}
            </span>
          </div>
        </div>

        {/* Column Content */}
        <div
          ref={drop}
          className={`p-4 min-h-96 ${isOver ? 'bg-opacity-75' : ''} transition-all duration-200`}
        >
          {tasks.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              <div className="w-12 h-12 bg-gray-200 rounded-full flex items-center justify-center mx-auto mb-3">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
              </div>
              <p className="text-sm">No tasks in {title.toLowerCase()}</p>
              {isOver && (
                <p className="text-xs mt-1 text-blue-600">Drop task here</p>
              )}
            </div>
          ) : (
            renderFlatView()
          )}
        </div>
      </div>
    </div>
  );
} 