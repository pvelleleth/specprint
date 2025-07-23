import React, { useState, useEffect } from 'react';
import { Task } from './types';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { GetWorkspaceBranches } from "../../../wailsjs/go/main/App";

interface BranchInfo {
  name: string;
  isRemote: boolean;
  isCurrent: boolean;
  hash?: string;
}

interface TaskEditModalProps {
  task: Task | null;
  allTasks: Task[];
  isOpen: boolean;
  onClose: () => void;
  onSave: (updatedTask: Task) => void;
  onRunTask?: (task: Task, baseBranch: string) => void;
  onAskForChanges?: (task: Task, prompt: string) => Promise<void>;
  selectedWorkspace?: any;
  isCreating?: boolean;
}

export function TaskEditModal({ 
  task, 
  allTasks, 
  isOpen, 
  onClose, 
  onSave, 
  onRunTask, 
  onAskForChanges,
  selectedWorkspace,
  isCreating = false 
}: TaskEditModalProps) {
  const [formData, setFormData] = useState<Partial<Task>>({});
  const [selectedDependencies, setSelectedDependencies] = useState<number[]>([]);
  const [branches, setBranches] = useState<BranchInfo[]>([]);
  const [selectedBranch, setSelectedBranch] = useState<string>('');
  const [loadingBranches, setLoadingBranches] = useState(false);
  const [branchError, setBranchError] = useState<string | null>(null);
  const [isChangesDialogOpen, setIsChangesDialogOpen] = useState(false);
  const [changesPrompt, setChangesPrompt] = useState('');
  const [isSubmittingChanges, setIsSubmittingChanges] = useState(false);

  // Initialize form data when task changes or when creating new task
  useEffect(() => {
    if (task && !isCreating) {
      setFormData({
        title: task.title,
        description: task.description,
        priority: task.priority,
        estimate: task.estimate,
      });
      setSelectedDependencies(task.dependencies || []);
    } else if (isCreating) {
      // Initialize with default values for new task
      setFormData({
        title: '',
        description: '',
        priority: 'medium',
        estimate: '',
      });
      setSelectedDependencies([]);
    }
  }, [task, isCreating]);

  // Load branches when modal opens and workspace is selected
  useEffect(() => {
    if (isOpen && selectedWorkspace && onRunTask) {
      loadBranches();
    }
  }, [isOpen, selectedWorkspace, onRunTask]);

  const loadBranches = async () => {
    if (!selectedWorkspace) return;
    
    setLoadingBranches(true);
    setBranchError(null);
    
    try {
      const result = await GetWorkspaceBranches(selectedWorkspace.name);
      if (result.success && result.branches) {
        setBranches(result.branches);
        // Default to current branch or first branch
        const currentBranch = result.branches.find(b => b.isCurrent);
        setSelectedBranch(currentBranch?.name || result.branches[0]?.name || '');
      } else {
        setBranchError(result.message || 'Failed to load branches');
      }
    } catch (error) {
      setBranchError('Failed to load branches');
      console.error('Error loading branches:', error);
    } finally {
      setLoadingBranches(false);
    }
  };

  const handleInputChange = (field: keyof Task, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleDependencyToggle = (taskId: number) => {
    setSelectedDependencies(prev => {
      if (prev.includes(taskId)) {
        return prev.filter(id => id !== taskId);
      } else {
        return [...prev, taskId];
      }
    });
  };

  const handleSave = () => {
    if (isCreating) {
      // Generate new ID for the task
      const maxId = Math.max(0, ...allTasks.map(t => t.id));
      const newTask: Task = {
        id: maxId + 1,
        title: formData.title || '',
        description: formData.description || '',
        priority: formData.priority || 'medium',
        estimate: formData.estimate || '',
        status: 'todo',
        dependencies: selectedDependencies,
      };
      onSave(newTask);
    } else if (task) {
      const updatedTask: Task = {
        ...task,
        ...formData,
        dependencies: selectedDependencies,
      };
      onSave(updatedTask);
    }
    onClose();
  };

  const handleRunTask = () => {
    if (!task || !onRunTask || !selectedBranch || isCreating) return;
    onRunTask(task, selectedBranch);
    onClose();
  };

  const handleAskForChanges = async () => {
    if (!task?.sessionId || !changesPrompt.trim() || !onAskForChanges) return;

    setIsSubmittingChanges(true);
    try {
      await onAskForChanges(task, changesPrompt);
      setIsChangesDialogOpen(false);
      setChangesPrompt('');
      onClose();
    } catch (error) {
      console.error('Error asking for changes:', error);
      alert('Error asking for changes: ' + error);
    } finally {
      setIsSubmittingChanges(false);
    }
  };

  const handleCancel = () => {
    onClose();
  };

  // Get available tasks for dependencies (exclude current task if editing)
  const availableTasks = allTasks.filter(t => !isCreating ? t.id !== task?.id : true);

  // Check if task can be run (only for existing tasks, not when creating)
  const canRunTask = !isCreating && task && onRunTask && task.status === 'todo' && !task.isRunning && selectedBranch;
  const hasDependencies = !isCreating && task?.dependencies && task.dependencies.length > 0;
  const dependencyTasks = !isCreating && task ? allTasks.filter(t => task.dependencies.includes(t.id)) : [];
  const completedDependencies = dependencyTasks.filter(t => t.status === 'done');
  const allDependenciesCompleted = hasDependencies && completedDependencies.length === (task?.dependencies.length || 0);
  const hasUncompletedDependencies = hasDependencies && !allDependenciesCompleted;

  // Add condition for showing the button - show for both done and in-progress tasks with sessionId
  const showAskForChanges = !isCreating && (task?.status === 'done' || task?.status === 'in-progress') && !!task?.sessionId && !!onAskForChanges;

  if (!isOpen || (!task && !isCreating)) {
    return null;
  }

  return (
    <>
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
        <div className="w-full max-w-2xl h-full max-h-[90vh] flex flex-col">
          <Card className="flex flex-col h-full bg-white shadow-xl">
            {/* Fixed Header */}
            <CardHeader className="flex-shrink-0 bg-white border-b">
              <CardTitle className="flex items-center gap-2">
                <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={isCreating 
                    ? "M12 6v6m0 0v6m0-6h6m-6 0H6" 
                    : "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"} />
                </svg>
                {isCreating ? 'Create New Task' : `Edit Task #${task?.id}`}
                {!isCreating && task?.isRunning && (
                  <span className="text-sm bg-blue-100 text-blue-800 px-2 py-1 rounded-full flex items-center gap-1">
                    <svg className="w-3 h-3 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    Running
                  </span>
                )}
              </CardTitle>
              <CardDescription>
                {isCreating 
                  ? 'Add a new task to your project'
                  : 'Update task details and manage dependencies'
                }
              </CardDescription>
            </CardHeader>
            
            {/* Scrollable Content */}
            <div className="flex-1 overflow-y-auto min-h-0">
              <CardContent className="space-y-6 p-6">
                {/* Basic Task Information */}
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Title <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      value={formData.title || ''}
                      onChange={(e) => handleInputChange('title', e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
                      placeholder="Enter task title"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Description
                    </label>
                    <textarea
                      value={formData.description || ''}
                      onChange={(e) => handleInputChange('description', e.target.value)}
                      rows={3}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white resize-none"
                      placeholder="Enter task description"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Priority
                      </label>
                      <select
                        value={formData.priority || 'medium'}
                        onChange={(e) => handleInputChange('priority', e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
                      >
                        <option value="low">Low</option>
                        <option value="medium">Medium</option>
                        <option value="high">High</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Estimate
                      </label>
                      <input
                        type="text"
                        value={formData.estimate || ''}
                        onChange={(e) => handleInputChange('estimate', e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
                        placeholder="e.g., 2h, 1d"
                      />
                    </div>
                  </div>
                </div>

                {/* Branch Selection Section - only show if onRunTask is available and not creating */}
                {!isCreating && onRunTask && (
                  <div className="border-t pt-6">
                    <h3 className="text-lg font-medium text-gray-900 mb-4 flex items-center gap-2">
                      <svg className="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      Branch Selection
                    </h3>
                    
                    <div className="space-y-3">
                      <p className="text-sm text-gray-600">
                        Select the base branch to create your task branch from:
                      </p>
                      
                      {loadingBranches ? (
                        <div className="flex items-center gap-2 text-sm text-gray-600">
                          <svg className="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                          Loading branches...
                        </div>
                      ) : branchError ? (
                        <div className="text-sm text-red-600 flex items-center gap-1">
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                        </svg>
                        {branchError}
                      </div>
                      ) : (
                        <div>
                          <select
                            value={selectedBranch}
                            onChange={(e) => setSelectedBranch(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
                          >
                            <option value="">Select a branch...</option>
                            {branches.map(branch => (
                              <option key={branch.name} value={branch.name}>
                                {branch.name}
                                {branch.isCurrent && ' (current)'}
                                {branch.isRemote && ' (remote)'}
                                {branch.hash && ` - ${branch.hash}`}
                              </option>
                            ))}
                          </select>
                          
                          {selectedBranch && (
                            <div className="mt-2 p-3 bg-green-50 border border-green-200 rounded-md">
                              <p className="text-sm text-green-800">
                                <strong>Selected:</strong> {selectedBranch}
                              </p>
                              <p className="text-xs text-green-600 mt-1">
                                Will create isolated worktree from this branch → run Claude Code → commit & push changes
                              </p>
                              <p className="text-xs text-green-600 mt-1">
                                <strong>✨ Benefit:</strong> Your main workspace stays untouched while task runs in parallel
                              </p>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Dependencies Section */}
                <div className="border-t pt-6">
                  <h3 className="text-lg font-medium text-gray-900 mb-4 flex items-center gap-2">
                    <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                    </svg>
                    Dependencies
                  </h3>
                  
                  {availableTasks.length === 0 ? (
                    <p className="text-gray-500 text-sm">No other tasks available for dependencies.</p>
                  ) : (
                    <div className="space-y-3">
                      <p className="text-sm text-gray-600">
                        Select tasks that must be completed before this task can start:
                      </p>
                      
                      <div className="max-h-40 overflow-y-auto border border-gray-200 rounded-md p-3 space-y-2 bg-white">
                        {availableTasks.map(availableTask => (
                          <label
                            key={availableTask.id}
                            className="flex items-center gap-3 p-2 hover:bg-gray-50 rounded cursor-pointer"
                          >
                            <input
                              type="checkbox"
                              checked={selectedDependencies.includes(availableTask.id)}
                              onChange={() => handleDependencyToggle(availableTask.id)}
                              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500 flex-shrink-0"
                            />
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <span className="font-medium text-sm text-gray-900">
                                  #{availableTask.id} {availableTask.title}
                                </span>
                                <span className={`text-xs px-2 py-1 rounded flex-shrink-0 ${
                                  availableTask.status === 'done' 
                                    ? 'bg-green-100 text-green-700' 
                                    : availableTask.status === 'in-progress'
                                    ? 'bg-yellow-100 text-yellow-700'
                                    : 'bg-blue-100 text-blue-700'
                                }`}>
                                  {availableTask.status}
                                </span>
                              </div>
                              <p className="text-xs text-gray-500 truncate">
                                {availableTask.description}
                              </p>
                            </div>
                          </label>
                        ))}
                      </div>
                      
                      {selectedDependencies.length > 0 && (
                        <div className="p-3 bg-blue-50 border border-blue-200 rounded-md">
                          <p className="text-sm text-blue-800 font-medium mb-2">
                            Selected Dependencies ({selectedDependencies.length}):
                          </p>
                          <div className="flex flex-wrap gap-1">
                            {selectedDependencies.map(depId => {
                              const depTask = availableTasks.find(t => t.id === depId);
                              return (
                                <span
                                  key={depId}
                                  className="text-xs px-2 py-1 bg-blue-100 text-blue-700 rounded border border-blue-200"
                                >
                                  #{depId} {depTask?.title}
                                </span>
                              );
                            })}
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </CardContent>
            </div>

            {/* Fixed Footer */}
            <div className="flex-shrink-0 p-6 border-t bg-gray-50">
              <div className="flex justify-between">
                {/* Left side - Run Task button (only for existing tasks) */}
                <div>
                  {canRunTask && !hasUncompletedDependencies && (
                    <Button
                      onClick={handleRunTask}
                      disabled={!selectedBranch || loadingBranches}
                      className="bg-green-600 hover:bg-green-700 text-white flex items-center gap-2"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.828 14.828a4 4 0 01-5.656 0M9 10h1m4 0h1m-6 4h1m4 0h1m6-10V9a3 3 0 01-3 3h-4m-3 0H7a3 3 0 01-3-3V4a3 3 0 013-3h4m3 0h4a3 3 0 013 3v5.172a4 4 0 00-1.172 2.828z" />
                      </svg>
                      {selectedBranch ? `Run Task (from ${selectedBranch})` : 'Run Task with Claude Code'}
                    </Button>
                  )}
                  {hasUncompletedDependencies && (
                    <div className="text-sm text-amber-600 flex items-center gap-1">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                      </svg>
                      Complete dependencies first to run task
                    </div>
                  )}
                  {!isCreating && task && onRunTask && task.status === 'todo' && !task.isRunning && !selectedBranch && (
                    <div className="text-sm text-amber-600 flex items-center gap-1">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
                      </svg>
                      Select a branch to run the task
                    </div>
                  )}
                  {!isCreating && task?.isRunning && (
                    <div className="text-sm text-blue-600 flex items-center gap-1">
                      <svg className="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                      </svg>
                      Task is currently running...
                    </div>
                  )}
                </div>

                {/* Right side - Save/Cancel/Ask for Changes buttons */}
                <div className="flex gap-3">
                  {showAskForChanges && (
                    <Button
                      onClick={() => setIsChangesDialogOpen(true)}
                      variant="secondary"
                      className="bg-purple-600 hover:bg-purple-700 text-white"
                    >
                      Ask for changes
                    </Button>
                  )}
                  <Button
                    onClick={handleCancel}
                    variant="outline"
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={handleSave}
                    disabled={!formData.title?.trim()}
                  >
                    {isCreating ? 'Create Task' : 'Save Changes'}
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* Ask for Changes Dialog */}
      {isChangesDialogOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[60] p-4">
          <div className="bg-white rounded-lg shadow-lg max-w-md w-full mx-4">
            <div className="px-6 py-4 border-b">
              <h2 className="text-lg font-semibold text-gray-900">
                Ask for Changes
              </h2>
            </div>
            <div className="p-6">
              <p className="text-sm text-gray-600 mb-4">
                Describe the changes you'd like Claude to make to this task:
              </p>
              <textarea
                value={changesPrompt}
                onChange={(e) => setChangesPrompt(e.target.value)}
                placeholder="e.g., Add error handling, improve the UI styling, add unit tests..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white resize-none"
                rows={4}
              />
              <div className="flex gap-3 mt-4">
                <Button
                  onClick={() => {
                    setIsChangesDialogOpen(false);
                    setChangesPrompt('');
                  }}
                  variant="outline"
                  className="flex-1"
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleAskForChanges}
                  disabled={!changesPrompt.trim() || isSubmittingChanges}
                  className="flex-1 bg-purple-600 hover:bg-purple-700 text-white"
                >
                  {isSubmittingChanges ? 'Submitting...' : 'Submit'}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
} 