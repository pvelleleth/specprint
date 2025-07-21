import { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { GetWorkspaces, OpenWorkspace } from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";

interface WorkspaceSidebarProps {
  selectedWorkspace: string | null;
  onWorkspaceSelect: (workspace: main.Workspace) => void;
  onNewWorkspace: () => void;
}

export function WorkspaceSidebar({ selectedWorkspace, onWorkspaceSelect, onNewWorkspace }: WorkspaceSidebarProps) {
  const [workspaces, setWorkspaces] = useState<main.Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
                    <CardTitle className="text-sm font-medium truncate">
                      {workspace.name}
                    </CardTitle>
                    <div className="flex items-center gap-1 ml-2">
                      {workspace.hasPrd && (
                        <div className="w-2 h-2 bg-green-500 rounded-full" title="Has PRD"></div>
                      )}
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
    </div>
  );
} 