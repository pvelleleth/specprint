import React, { useState, useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { EnhancedKanbanColumn } from './EnhancedKanbanColumn';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { GenerateTasksFromWorkspacePRD } from "../../../wailsjs/go/main/App";
import { main } from "../../../wailsjs/go/models";
import { Task, BoardState } from './types';

interface KanbanBoardProps {
  selectedWorkspace: main.Workspace | null;
}

export function KanbanBoard({ selectedWorkspace }: KanbanBoardProps) {
  const [boardState, setBoardState] = useState<BoardState>({ 
    tasks: [], 
    lastUpdated: '' 
  });
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationResult, setGenerationResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Load board state from localStorage on component mount
  useEffect(() => {
    if (selectedWorkspace) {
      const saved = localStorage.getItem(`kanban-${selectedWorkspace.name}`);
      if (saved) {
        try {
          const parsed = JSON.parse(saved);
          // Ensure backward compatibility with old board state format
          const migratedState: BoardState = {
            tasks: parsed.tasks || [],
            lastUpdated: parsed.lastUpdated || ''
          };
          
          // Remove storyId migration
          setBoardState(migratedState);
        } catch (err) {
          console.error('Failed to parse saved board state:', err);
          // Reset to default state if parsing fails
          setBoardState({ 
            tasks: [], 
            lastUpdated: '' 
          });
        }
      }
    }
  }, [selectedWorkspace]);

  // Save board state to localStorage whenever it changes
  useEffect(() => {
    if (selectedWorkspace && boardState.tasks && boardState.tasks.length > 0) {
      localStorage.setItem(`kanban-${selectedWorkspace.name}`, JSON.stringify(boardState));
    }
  }, [boardState, selectedWorkspace]);

  const generateTasksFromPRD = async () => {
    if (!selectedWorkspace) {
      setError('No workspace selected');
      return;
    }

    if (!selectedWorkspace.hasPrd) {
      setError('Selected workspace does not have a PRD. Please create a PRD first.');
      return;
    }

    setIsGenerating(true);
    setError(null);
    setGenerationResult(null);

    try {
      const result = await GenerateTasksFromWorkspacePRD(selectedWorkspace.name);
      
      if (result.success && result.tasks) { // Changed from epics
        const newTasks = result.tasks.map(task => ({
          ...task,
          status: 'todo'
        }));

        const newBoardState: BoardState = {
          tasks: newTasks,
          lastUpdated: new Date().toISOString(),
        };

        setBoardState(newBoardState);
        setGenerationResult(`Successfully generated ${newTasks.length} tasks from PRD`);
      } else {
        setError(result.message || 'Failed to generate tasks from PRD');
      }
    } catch (err) {
      setError(`Error generating tasks: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsGenerating(false);
    }
  };

  const handleTaskMove = (task: Task, sourceColumnId: string, targetColumnId: string) => {
    if (sourceColumnId === targetColumnId) return;

    // Check if task has uncompleted dependencies
    const hasDependencies = task.dependencies && task.dependencies.length > 0;
    if (hasDependencies) {
      const dependencyTasks = boardState.tasks.filter(t => task.dependencies.includes(t.id));
      const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
      const allDependenciesCompleted = completedDependencies.length === task.dependencies.length;
      
      if (!allDependenciesCompleted && targetColumnId !== 'todo') {
        // Show error message for blocked tasks
        setError(`Cannot move task #${task.id} to ${targetColumnId}. Complete dependencies first: ${task.dependencies.join(', ')}`);
        return;
      }
    }

    // Map column IDs to status values
    const statusMap: { [key: string]: string } = {
      'todo': 'todo',
      'in-progress': 'in-progress',
      'done': 'done'
    };

    const updatedTasks = boardState.tasks.map(t => 
      t.id === task.id 
        ? { ...t, status: statusMap[targetColumnId] || 'todo' }
        : t
    );

    setBoardState({
      ...boardState,
      tasks: updatedTasks,
      lastUpdated: new Date().toISOString(),
    });

    // Clear any previous error messages
    setError(null);
  };

  const clearBoard = () => {
    setBoardState({ 
      tasks: [], 
      lastUpdated: '' 
    });
    if (selectedWorkspace) {
      localStorage.removeItem(`kanban-${selectedWorkspace.name}`);
    }
    setGenerationResult(null);
    setError(null);
  };

  // Group tasks by status for display in columns
  const todoTasks = (boardState.tasks || []).filter(task => task.status === 'todo');
  const inProgressTasks = (boardState.tasks || []).filter(task => task.status === 'in-progress');
  const doneTasks = (boardState.tasks || []).filter(task => task.status === 'done');

  const getTaskCounts = () => {
    const total = (boardState.tasks || []).length;
    const completed = doneTasks.length;
    const inProgress = inProgressTasks.length;
    const remaining = todoTasks.length;
    
    return { total, completed, inProgress, remaining };
  };

  const taskCounts = getTaskCounts();

  // Calculate dependency statistics
  const getDependencyStats = () => {
    const tasksWithDependencies = boardState.tasks.filter(task => task.dependencies && task.dependencies.length > 0);
    const blockedTasks = boardState.tasks.filter(task => {
      if (!task.dependencies || task.dependencies.length === 0) return false;
      const dependencyTasks = boardState.tasks.filter(t => task.dependencies.includes(t.id));
      const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
      return completedDependencies.length < task.dependencies.length;
    });
    
    return {
      totalTasks: boardState.tasks.length,
      tasksWithDependencies: tasksWithDependencies.length,
      blockedTasks: blockedTasks.length,
      readyTasks: boardState.tasks.length - blockedTasks.length
    };
  };

  const dependencyStats = getDependencyStats();

  if (!selectedWorkspace) {
    return (
      <Card className="w-full max-w-4xl mx-auto">
        <CardHeader>
          <CardTitle>Kanban Board</CardTitle>
          <CardDescription>
            Please select a workspace from the sidebar to view or generate tasks.
          </CardDescription>
        </CardHeader>
        <CardContent className="text-center py-12">
          <div className="max-w-md mx-auto">
            <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 002 2m0 0v10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2h2a2 2 0 002-2" />
              </svg>
            </div>
            <h3 className="text-lg font-medium mb-2">No Workspace Selected</h3>
            <p className="text-muted-foreground">
              Select a workspace to view your project tasks and Kanban board.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="w-full space-y-6">
      {/* Header Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 002 2m0 0v10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2h2a2 2 0 002-2" />
                </svg>
                Kanban Board
              </CardTitle>
                             <CardDescription>
                 Project tasks for {selectedWorkspace.name}
               </CardDescription>
            </div>
            
            <div className="flex items-center gap-3">
              {boardState.tasks && boardState.tasks.length > 0 && (
                <div className="text-sm text-gray-600 text-right">
                  <div className="font-medium">
                    {taskCounts.completed}/{taskCounts.total} tasks completed
                  </div>
                  <div className="text-xs">
                    {taskCounts.inProgress} in progress • {taskCounts.remaining} remaining
                  </div>
                  {dependencyStats.tasksWithDependencies > 0 && (
                    <div className="text-xs mt-1">
                      <span className="text-green-600">{dependencyStats.readyTasks} ready</span> • 
                      <span className="text-amber-600"> {dependencyStats.blockedTasks} blocked</span>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </CardHeader>
        
        <CardContent>
          <div className="flex flex-wrap gap-3">
            <Button 
              onClick={generateTasksFromPRD}
              disabled={isGenerating || !selectedWorkspace.hasPrd}
              className="flex items-center gap-2"
            >
              {isGenerating ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                  Generating Tasks...
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                  {boardState.tasks && boardState.tasks.length > 0 ? 'Regenerate from PRD' : 'Generate from PRD'}
                </>
              )}
            </Button>

            {boardState.tasks && boardState.tasks.length > 0 && (
              <>
                <Button 
                  onClick={clearBoard}
                  variant="outline"
                  className="flex items-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Clear Board
                </Button>
              </>
            )}
          </div>

          {!selectedWorkspace.hasPrd && (
            <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
              <div className="flex items-center gap-2">
                <svg className="w-4 h-4 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.664-.833-2.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
                <p className="text-sm text-yellow-800 font-medium">
                  This workspace doesn't have a PRD yet. Create a PRD first to generate tasks.
                </p>
              </div>
            </div>
          )}

          {/* Status Messages */}
          {error && (
            <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
              <div className="flex items-start gap-2">
                <svg className="w-5 h-5 text-red-500 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div>
                  <p className="font-medium text-red-800">Error</p>
                  <p className="text-sm text-red-700 mt-1">{error}</p>
                </div>
              </div>
            </div>
          )}

          {generationResult && (
            <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-lg">
              <div className="flex items-start gap-2">
                <svg className="w-5 h-5 text-green-500 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <div>
                  <p className="font-medium text-green-800">Success!</p>
                  <p className="text-sm text-green-700 mt-1">{generationResult}</p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Dependency Overview */}
      {boardState.tasks && boardState.tasks.length > 0 && dependencyStats.tasksWithDependencies > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
              </svg>
              Dependency Overview
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center p-3 bg-blue-50 rounded-lg">
                <div className="text-2xl font-bold text-blue-600">{dependencyStats.totalTasks}</div>
                <div className="text-sm text-blue-700">Total Tasks</div>
              </div>
              <div className="text-center p-3 bg-green-50 rounded-lg">
                <div className="text-2xl font-bold text-green-600">{dependencyStats.readyTasks}</div>
                <div className="text-sm text-green-700">Ready to Start</div>
              </div>
              <div className="text-center p-3 bg-amber-50 rounded-lg">
                <div className="text-2xl font-bold text-amber-600">{dependencyStats.blockedTasks}</div>
                <div className="text-sm text-amber-700">Blocked</div>
              </div>
              <div className="text-center p-3 bg-purple-50 rounded-lg">
                <div className="text-2xl font-bold text-purple-600">{dependencyStats.tasksWithDependencies}</div>
                <div className="text-sm text-purple-700">Have Dependencies</div>
              </div>
            </div>
            
            {dependencyStats.blockedTasks > 0 && (
              <div className="mt-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <svg className="w-4 h-4 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                  </svg>
                  <span className="font-medium text-amber-800">Blocked Tasks</span>
                </div>
                <p className="text-sm text-amber-700">
                  {dependencyStats.blockedTasks} tasks are blocked by incomplete dependencies. 
                  Complete the prerequisite tasks to unlock them.
                </p>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Kanban Board */}
      {boardState.tasks && boardState.tasks.length > 0 && (
        <DndProvider backend={HTML5Backend}>
          <div className="flex gap-6 overflow-x-auto pb-4">
            <EnhancedKanbanColumn
              title="To Do"
              columnId="todo"
              tasks={todoTasks}
              allTasks={boardState.tasks}
              onTaskMove={handleTaskMove}
            />
            <EnhancedKanbanColumn
              title="In Progress"
              columnId="in-progress"
              tasks={inProgressTasks}
              allTasks={boardState.tasks}
              onTaskMove={handleTaskMove}
            />
            <EnhancedKanbanColumn
              title="Done"
              columnId="done"
              tasks={doneTasks}
              allTasks={boardState.tasks}
              onTaskMove={handleTaskMove}
            />
          </div>
        </DndProvider>
      )}

      {/* Empty State */}
      {(!boardState.tasks || boardState.tasks.length === 0) && !isGenerating && (
        <Card>
          <CardContent className="text-center py-12">
            <div className="max-w-md mx-auto">
              <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
                </svg>
              </div>
              <h3 className="text-lg font-medium mb-2">No Tasks Yet</h3>
              <p className="text-muted-foreground mb-4">
                {selectedWorkspace.hasPrd 
                  ? 'Generate tasks from your PRD to get started with project management.'
                  : 'Create a PRD first, then generate tasks to populate your Kanban board.'
                }
              </p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
} 