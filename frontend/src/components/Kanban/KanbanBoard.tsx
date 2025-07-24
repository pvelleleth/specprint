import React, { useState, useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { EnhancedKanbanColumn } from './EnhancedKanbanColumn';
import { TaskEditModal } from './TaskEditModal';
import { GenerateTasksFromWorkspacePRD, StartTaskConversation, CleanupTaskWorktree, ContinueClaudeSession } from "../../../wailsjs/go/main/App";
import { Task, BoardState } from './types';

interface KanbanBoardProps {
  selectedWorkspace: any;
}

export function KanbanBoard({ selectedWorkspace }: KanbanBoardProps) {
  const [boardState, setBoardState] = useState<BoardState>({ 
    tasks: [], 
    lastUpdated: '' 
  });
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationResult, setGenerationResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [runTaskResult, setRunTaskResult] = useState<string | null>(null);

  // Load board state from localStorage on component mount
  useEffect(() => {
    if (selectedWorkspace) {
      const saved = localStorage.getItem(`kanban-${selectedWorkspace.name}`);
      if (saved) {
        try {
          const parsed = JSON.parse(saved);
          const migratedState: BoardState = {
            tasks: parsed.tasks || [],
            lastUpdated: parsed.lastUpdated || ''
          };
          setBoardState(migratedState);
        } catch (err) {
          console.error('Failed to parse saved board state:', err);
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

  const generateTasks = async () => {
    if (!selectedWorkspace) {
      setError('No workspace selected');
      return;
    }

    setIsGenerating(true);
    setError(null);
    setGenerationResult(null);

    try {
      const result = await GenerateTasksFromWorkspacePRD(selectedWorkspace.name);
      
      if (result.success && result.tasks) {
        const tasksWithStatus = result.tasks.map((task: any) => ({
          ...task,
          status: 'todo',
          isRunning: false // Initialize isRunning state
        }));

        setBoardState({
          tasks: tasksWithStatus,
          lastUpdated: new Date().toISOString(),
        });

        setGenerationResult(`Successfully generated ${result.tasks.length} tasks from PRD!`);
      } else {
        setError(result.message || 'Failed to generate tasks');
      }
    } catch (err) {
      console.error('Error generating tasks:', err);
      setError('An unexpected error occurred while generating tasks');
    } finally {
      setIsGenerating(false);
    }
  };

  const handleTaskMove = async (task: Task, sourceColumnId: string, targetColumnId: string) => {
    if (sourceColumnId === targetColumnId) return;

    const hasDependencies = task.dependencies && task.dependencies.length > 0;
    if (hasDependencies) {
      const dependencyTasks = boardState.tasks.filter(t => task.dependencies.includes(t.id));
      const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
      const allDependenciesCompleted = completedDependencies.length === task.dependencies.length;
      
      if (!allDependenciesCompleted && targetColumnId !== 'todo') {
        setError(`Cannot move task #${task.id} to ${targetColumnId}. Complete dependencies first: ${task.dependencies.join(', ')}`);
        return;
      }
    }

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

    setError(null);

    if (targetColumnId === 'done') {
      try {
        await CleanupTaskWorktree(selectedWorkspace.name, task.id);
        console.log(`Worktree for task ${task.id} cleaned up.`);
        
        // Update task status and clear worktree path after successful cleanup
        const updatedTasks = boardState.tasks.map(t => 
          t.id === task.id 
            ? { ...t, status: targetColumnId, worktreePath: undefined }
            : t
        );

        setBoardState({
          ...boardState,
          tasks: updatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      } catch (err) {
        console.error(`Error cleaning up worktree for task ${task.id}:`, err);
        setError(`Failed to clean up worktree for task ${task.id}`);
        
        // Still update the status even if cleanup failed
        const updatedTasks = boardState.tasks.map(t => 
          t.id === task.id 
            ? { ...t, status: targetColumnId }
            : t
        );

        setBoardState({
          ...boardState,
          tasks: updatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      }
    } else {
      // Normal task move without cleanup
      const updatedTasks = boardState.tasks.map(t => 
        t.id === task.id 
          ? { ...t, status: targetColumnId }
          : t
      );

      setBoardState({
        ...boardState,
        tasks: updatedTasks,
        lastUpdated: new Date().toISOString(),
      });
    }
  };

  // Handle task status updates with worktree cleanup
  const handleTaskStatusUpdate = async (taskId: number, newStatus: string) => {
    // Clean up worktree if task is marked as done
    if (newStatus === 'done' && selectedWorkspace) {
      try {
        await CleanupTaskWorktree(selectedWorkspace.name, taskId);
        console.log(`Worktree for task ${taskId} cleaned up after marking as done.`);
        
        // Update task status and clear worktree path after successful cleanup
        const updatedTasks = boardState.tasks.map(t => 
          t.id === taskId 
            ? { ...t, status: newStatus, worktreePath: undefined }
            : t
        );

        setBoardState({
          ...boardState,
          tasks: updatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      } catch (err) {
        console.error(`Error cleaning up worktree for task ${taskId}:`, err);
        setError(`Task marked as done but failed to clean up worktree: ${err}`);
        
        // Still update the status even if cleanup failed
        const updatedTasks = boardState.tasks.map(t => 
          t.id === taskId 
            ? { ...t, status: newStatus }
            : t
        );

        setBoardState({
          ...boardState,
          tasks: updatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      }
    } else {
      // Normal status update without cleanup
      const updatedTasks = boardState.tasks.map(t => 
        t.id === taskId 
          ? { ...t, status: newStatus }
          : t
      );

      setBoardState({
        ...boardState,
        tasks: updatedTasks,
        lastUpdated: new Date().toISOString(),
      });
    }
  };

  const handleEditTask = (task: Task) => {
    setEditingTask(task);
    setIsEditModalOpen(true);
  };

  const handleSaveTask = (updatedTask: Task) => {
    if (editingTask) {
      // Editing existing task
      const updatedTasks = boardState.tasks.map(t => 
        t.id === updatedTask.id ? updatedTask : t
      );

      setBoardState({
        ...boardState,
        tasks: updatedTasks,
        lastUpdated: new Date().toISOString(),
      });

      setEditingTask(null);
      setIsEditModalOpen(false);
    } else {
      // Creating new task
      setBoardState({
        ...boardState,
        tasks: [...boardState.tasks, updatedTask],
        lastUpdated: new Date().toISOString(),
      });

      setIsCreateModalOpen(false);
    }
  };

  const handleCloseEditModal = () => {
    setEditingTask(null);
    setIsEditModalOpen(false);
  };

  const handleOpenCreateModal = () => {
    setIsCreateModalOpen(true);
  };

  const handleCloseCreateModal = () => {
    setIsCreateModalOpen(false);
  };

  const handleRunTask = async (task: Task, baseBranch: string) => {
    if (!selectedWorkspace) {
      setError('No workspace selected');
      return;
    }

    if (!baseBranch) {
      setError('Base branch must be selected');
      return;
    }

    const hasDependencies = task.dependencies && task.dependencies.length > 0;
    if (hasDependencies) {
      const dependencyTasks = boardState.tasks.filter(t => task.dependencies.includes(t.id));
      const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
      const allDependenciesCompleted = completedDependencies.length === task.dependencies.length;
      
      if (!allDependenciesCompleted) {
        setError(`Cannot run task #${task.id}. Complete dependencies first: ${task.dependencies.join(', ')}`);
        return;
      }
    }

    setError(null);
    setRunTaskResult(null);

    // Set the task as running
    const updatedTasks = boardState.tasks.map(t => 
      t.id === task.id 
        ? { ...t, isRunning: true }
        : t
    );

    setBoardState({
      ...boardState,
      tasks: updatedTasks,
      lastUpdated: new Date().toISOString(),
    });

    try {
      const result = await StartTaskConversation(selectedWorkspace.name, task.id, task.title, task.description, baseBranch);
      
      if (result.success) {
        setRunTaskResult(`Successfully started task conversation ${task.id} on branch '${result.branchName}' (based on '${baseBranch}'). ${result.message}`);
        
        // Update task to in-progress and stop running state, add worktree path and session details
        const finalUpdatedTasks = boardState.tasks.map(t => 
          t.id === task.id 
            ? { 
                ...t, 
                status: 'in-progress', 
                isRunning: false, 
                worktreePath: result.worktreePath || `task-${task.id}-${selectedWorkspace.name}`,
                sessionId: result.sessionId
              }
            : t
        );

        setBoardState({
          ...boardState,
          tasks: finalUpdatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      } else {
        setError(result.message || 'Failed to start task conversation');
        // Just stop running state on failure
        const failedUpdatedTasks = boardState.tasks.map(t => 
          t.id === task.id 
            ? { ...t, isRunning: false }
            : t
        );
        setBoardState({
          ...boardState,
          tasks: failedUpdatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      }
    } catch (err) {
      console.error('Error starting task conversation:', err);
      setError('An unexpected error occurred while starting the task conversation');
      // Stop running state on error
      const errorUpdatedTasks = boardState.tasks.map(t => 
        t.id === task.id 
          ? { ...t, isRunning: false }
          : t
      );
      setBoardState({
        ...boardState,
        tasks: errorUpdatedTasks,
        lastUpdated: new Date().toISOString(),
      });
    }
  };

  const handleAskForChanges = async (task: Task, prompt: string) => {
    if (!task.sessionId || !selectedWorkspace) {
      throw new Error('No session ID available for this task');
    }

    if (!task.worktreePath) {
      throw new Error('No worktree path available for this task');
    }

    try {
      // Use the ContinueClaudeSession function with sessionId, userMessage, and worktreePath
      const result = await ContinueClaudeSession(task.sessionId, prompt, task.worktreePath);
      
      if (result.success) {
        // Update task status to in-progress if it was done
        const updatedTasks = boardState.tasks.map(t => 
          t.id === task.id 
            ? { ...t, status: t.status === 'done' ? 'in-progress' : t.status }
            : t
        );

        setBoardState({
          ...boardState,
          tasks: updatedTasks,
          lastUpdated: new Date().toISOString(),
        });
      } else {
        throw new Error(result.message);
      }
    } catch (error) {
      console.error('Error continuing Claude session:', error);
      throw error;
    }
  };

  const clearBoard = () => {
    setBoardState({
      tasks: [],
      lastUpdated: new Date().toISOString(),
    });
    setGenerationResult(null);
    setError(null);
    setRunTaskResult(null);
  };

  // Calculate statistics
  const totalTasks = boardState.tasks.length;
  const tasksWithDependencies = boardState.tasks.filter(task => task.dependencies && task.dependencies.length > 0);
  const readyTasks = boardState.tasks.filter(task => {
    if (!task.dependencies || task.dependencies.length === 0) return true;
    const dependencyTasks = boardState.tasks.filter(t => task.dependencies.includes(t.id));
    const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
    return completedDependencies.length === task.dependencies.length;
  });
  const blockedTasks = boardState.tasks.filter(task => {
    if (!task.dependencies || task.dependencies.length === 0) return false;
    const dependencyTasks = boardState.tasks.filter(t => task.dependencies.includes(t.id));
    const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
    return completedDependencies.length !== task.dependencies.length;
  });

  if (!selectedWorkspace) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-gray-500 text-lg">Select a workspace to view its Kanban board</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b border-gray-200 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">
              {selectedWorkspace.name} - Kanban Board
            </h2>
            {boardState.tasks.length > 0 && (
              <div className="mt-2 flex items-center gap-4 text-sm text-gray-600">
                <span>
                  {boardState.tasks.filter(t => t.status === 'done').length} / {totalTasks} completed
                </span>
                <span>
                  {boardState.tasks.filter(t => t.status === 'in-progress').length} in progress
                </span>
                <span>
                  {boardState.tasks.filter(t => t.status === 'todo').length} remaining
                </span>
                {tasksWithDependencies.length > 0 && (
                  <>
                    <span className="text-green-600">
                      {readyTasks.length} ready
                    </span>
                    <span className="text-red-600">
                      {blockedTasks.length} blocked
                    </span>
                  </>
                )}
              </div>
            )}
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleOpenCreateModal}
              className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 flex items-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              Add Task
            </button>
            <button
              onClick={generateTasks}
              disabled={isGenerating || !selectedWorkspace?.hasPrd}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isGenerating ? 'Generating...' : 'Generate from PRD'}
            </button>
            <button
              onClick={clearBoard}
              className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700"
            >
              Clear Board
            </button>
          </div>
        </div>

        {!selectedWorkspace?.hasPrd && (
          <div className="mt-2 p-2 bg-yellow-100 border border-yellow-300 rounded text-yellow-800 text-sm">
            This workspace doesn't have a PRD file. Create one first to generate tasks.
          </div>
        )}
      </div>

      {/* Status Messages */}
      {(error || generationResult || runTaskResult) && (
        <div className="p-4 border-b border-gray-200">
          {error && (
            <div className="p-3 bg-red-100 border border-red-300 rounded text-red-800 text-sm mb-2">
              {error}
              <button
                onClick={() => setError(null)}
                className="ml-2 text-red-600 hover:text-red-800"
              >
                ✕
              </button>
            </div>
          )}
          
          {generationResult && (
            <div className="p-3 bg-green-100 border border-green-300 rounded text-green-800 text-sm mb-2">
              {generationResult}
              <button
                onClick={() => setGenerationResult(null)}
                className="ml-2 text-green-600 hover:text-green-800"
              >
                ✕
              </button>
            </div>
          )}

          {runTaskResult && (
            <div className="p-3 bg-blue-100 border border-blue-300 rounded text-blue-800 text-sm">
              {runTaskResult}
              <button
                onClick={() => setRunTaskResult(null)}
                className="ml-2 text-blue-600 hover:text-blue-800"
              >
                ✕
              </button>
            </div>
          )}
        </div>
      )}

      {/* Dependency Overview */}
      {totalTasks > 0 && tasksWithDependencies.length > 0 && (
        <div className="p-4 border-b border-gray-200 bg-gray-50">
          <h3 className="text-sm font-medium text-gray-700 mb-2">Dependency Overview</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div className="bg-white p-2 rounded border">
              <div className="font-medium text-gray-900">{totalTasks}</div>
              <div className="text-gray-600">Total Tasks</div>
            </div>
            <div className="bg-white p-2 rounded border">
              <div className="font-medium text-green-600">{readyTasks.length}</div>
              <div className="text-gray-600">Ready to Start</div>
            </div>
            <div className="bg-white p-2 rounded border">
              <div className="font-medium text-red-600">{blockedTasks.length}</div>
              <div className="text-gray-600">Blocked</div>
            </div>
            <div className="bg-white p-2 rounded border">
              <div className="font-medium text-blue-600">{tasksWithDependencies.length}</div>
              <div className="text-gray-600">Have Dependencies</div>
            </div>
          </div>
          {blockedTasks.length > 0 && (
            <div className="mt-2 text-xs text-amber-600">
              ⚠️ {blockedTasks.length} task(s) are waiting for dependencies to be completed
            </div>
          )}
        </div>
      )}

      {/* Kanban Board */}
      <div className="flex-1 overflow-hidden">
        {boardState.tasks.length === 0 ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <p className="text-gray-500 text-lg mb-4">No tasks generated yet</p>
              {selectedWorkspace?.hasPrd ? (
                <p className="text-gray-400">Click "Generate from PRD" to create tasks from your PRD</p>
              ) : (
                <p className="text-gray-400">Create a PRD first, then generate tasks</p>
              )}
            </div>
          </div>
        ) : (
          <DndProvider backend={HTML5Backend}>
            <div className="h-full flex gap-6 p-6 overflow-x-auto">
              <EnhancedKanbanColumn
                title="To Do"
                tasks={boardState.tasks.filter(task => task.status === 'todo')}
                columnId="todo"
                onTaskMove={handleTaskMove}
                allTasks={boardState.tasks}
                onEditTask={handleEditTask}
                onRunTask={handleRunTask}
              />
              <EnhancedKanbanColumn
                title="In Progress"
                tasks={boardState.tasks.filter(task => task.status === 'in-progress')}
                columnId="in-progress"
                onTaskMove={handleTaskMove}
                allTasks={boardState.tasks}
                onEditTask={handleEditTask}
                onRunTask={handleRunTask}
              />
              <EnhancedKanbanColumn
                title="Done"
                tasks={boardState.tasks.filter(task => task.status === 'done')}
                columnId="done"
                onTaskMove={handleTaskMove}
                allTasks={boardState.tasks}
                onEditTask={handleEditTask}
                onRunTask={handleRunTask}
              />
            </div>
          </DndProvider>
        )}
      </div>

      {/* Task Edit Modal */}
      <TaskEditModal
        task={editingTask}
        allTasks={boardState.tasks}
        isOpen={isEditModalOpen}
        onClose={handleCloseEditModal}
        onSave={handleSaveTask}
        onRunTask={handleRunTask}
        onAskForChanges={handleAskForChanges}
        selectedWorkspace={selectedWorkspace}
      />

      {/* Task Create Modal */}
      <TaskEditModal
        task={null}
        allTasks={boardState.tasks}
        isOpen={isCreateModalOpen}
        onClose={handleCloseCreateModal}
        onSave={handleSaveTask}
        selectedWorkspace={selectedWorkspace}
        isCreating={true}
      />
    </div>
  );
} 