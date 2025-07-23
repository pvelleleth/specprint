import { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { GetWorkspaces, OpenWorkspace, DeleteWorkspace } from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";

interface WorkspaceSidebarProps {
  selectedWorkspace: string | null;
  onWorkspaceSelect: (workspace: main.Workspace) => void;
  onNewWorkspace: () => void;
}

interface DeleteConfirmation {
  workspace: main.Workspace;
  isOpen: boolean;
}

export function WorkspaceSidebar({ selectedWorkspace, onWorkspaceSelect, onNewWorkspace }: WorkspaceSidebarProps) {
  const [workspaces, setWorkspaces] = useState<main.Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteConfirmation, setDeleteConfirmation] = useState<DeleteConfirmation>({ workspace: null as any, isOpen: false });
  const [deleting, setDeleting] = useState<string | null>(null);

  const loadWorkspaces = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await GetWorkspaces();
      
      if (result.success && result.workspaces) {
        // Sort workspaces by last opened (most recent first)
        const sortedWorkspaces = result.workspaces.sort((a, b) => {
          const aTime = new Date(a.lastOpened).getTime();
          const bTime = new Date(b.lastOpened).getTime();
          return bTime - aTime;
        });
        setWorkspaces(sortedWorkspaces);
      } else {
        setError(result.message || 'Failed to load workspaces');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error occurred');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadWorkspaces();
  }, []);

  const handleWorkspaceClick = async (workspace: main.Workspace) => {
    try {
      // Update last opened time
      await OpenWorkspace(workspace.name);
      onWorkspaceSelect(workspace);
      // Reload workspaces to update the order
      loadWorkspaces();
    } catch (err) {
      console.error('Failed to open workspace:', err);
    }
  };

  const handleDeleteClick = (workspace: main.Workspace, event: React.MouseEvent) => {
    event.stopPropagation(); // Prevent workspace selection
    setDeleteConfirmation({ workspace, isOpen: true });
  };

  const handleDeleteConfirm = async (deleteFiles: boolean) => {
    if (!deleteConfirmation.workspace) return;
    
    const workspaceName = deleteConfirmation.workspace.name;
    setDeleting(workspaceName);
    
    try {
      const result = await DeleteWorkspace(workspaceName, deleteFiles);
      
      if (result.success) {
        // If the deleted workspace was selected, clear selection
        if (selectedWorkspace === workspaceName) {
          onWorkspaceSelect(null as any);
        }
        
        // Reload workspaces
        await loadWorkspaces();
        
        // Show success message (you could replace this with a toast notification)
        alert(result.message);
      } else {
        setError(result.message);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete workspace');
    } finally {
      setDeleting(null);
      setDeleteConfirmation({ workspace: null as any, isOpen: false });
    }
  };

  const handleDeleteCancel = () => {
    setDeleteConfirmation({ workspace: null as any, isOpen: false });
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffHours / 24);

    if (diffHours < 1) return 'Just now';
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="w-80 h-full bg-muted/20 border-r border-border flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-border">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold">Workspaces</h2>
          <Button 
            onClick={onNewWorkspace}
            size="sm"
            variant="outline"
          >
            + New
          </Button>
        </div>
        <p className="text-sm text-muted-foreground">
          Select a workspace to manage its PRD
        </p>
      </div>

      {/* Workspaces List */}
      <div className="flex-1 overflow-y-auto p-4">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary"></div>
          </div>
        ) : error ? (
          <div className="text-center py-8">
            <p className="text-sm text-red-600 mb-3">{error}</p>
            <Button onClick={loadWorkspaces} size="sm" variant="outline">
              Retry
            </Button>
          </div>
        ) : workspaces.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-sm text-muted-foreground mb-3">
              No workspaces found
            </p>
            <p className="text-xs text-muted-foreground mb-4">
              Clone a repository to get started
            </p>
            <Button onClick={onNewWorkspace} size="sm">
              Clone Repository
            </Button>
          </div>
        ) : (
          <div className="space-y-2">
            {workspaces.map((workspace) => (
              <Card 
                key={workspace.name}
                className={`cursor-pointer transition-colors hover:bg-accent/50 ${
                  selectedWorkspace === workspace.name 
                    ? 'bg-accent border-primary' 
                    : 'hover:border-accent-foreground/20'
                }`}
                onClick={() => handleWorkspaceClick(workspace)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between">
                    <CardTitle className="text-sm font-medium truncate flex-1 mr-2">
                      {workspace.name}
                    </CardTitle>
                    <div className="flex items-center gap-1">
                      {workspace.hasPrd && (
                        <div className="w-2 h-2 bg-green-500 rounded-full" title="Has PRD"></div>
                      )}
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-6 w-6 p-0 text-red-500 hover:text-red-700 hover:bg-red-100"
                        onClick={(e) => handleDeleteClick(workspace, e)}
                        disabled={deleting === workspace.name}
                        title="Delete workspace"
                      >
                        {deleting === workspace.name ? (
                          <div className="animate-spin rounded-full h-3 w-3 border border-red-500 border-t-transparent"></div>
                        ) : (
                          <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        )}
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-0">
                  <div className="space-y-1">
                    <p className="text-xs text-muted-foreground truncate">
                      {workspace.repoUrl}
                    </p>
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <span>Last opened</span>
                      <span>{formatDate(workspace.lastOpened)}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-border">
        <Button 
          onClick={loadWorkspaces}
          variant="ghost" 
          size="sm" 
          className="w-full"
        >
          Refresh Workspaces
        </Button>
      </div>

      {/* Delete Confirmation Modal */}
      {deleteConfirmation.isOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
            <h3 className="text-lg font-semibold mb-4">Delete Workspace</h3>
            <p className="text-gray-600 mb-4">
              Are you sure you want to delete <strong>{deleteConfirmation.workspace?.name}</strong>?
            </p>
            <p className="text-sm text-gray-500 mb-6">
              Choose whether to keep or delete the repository files:
            </p>
            
            <div className="flex flex-col gap-3">
              <Button
                onClick={() => handleDeleteConfirm(false)}
                variant="outline"
                className="w-full"
                disabled={deleting !== null}
              >
                Remove from List Only
                <span className="text-xs text-gray-500 block">Keep files at {deleteConfirmation.workspace?.path}</span>
              </Button>
              
              <Button
                onClick={() => handleDeleteConfirm(true)}
                variant="destructive"
                className="w-full"
                disabled={deleting !== null}
              >
                Delete Completely
                <span className="text-xs text-red-200 block">Remove all files permanently</span>
              </Button>
              
              <Button
                onClick={handleDeleteCancel}
                variant="ghost"
                className="w-full"
                disabled={deleting !== null}
              >
                Cancel
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
} 