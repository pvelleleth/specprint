import {useState} from 'react';
import {Greet} from "../wailsjs/go/main/App";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { GitRepositoryCloner } from "@/components/GitRepositoryCloner";
import { PRDInput } from "@/components/PRDInput";
import { WorkspaceSidebar } from "@/components/WorkspaceSidebar";
import { KanbanBoard } from "@/components/Kanban";
import { main } from "../wailsjs/go/models";

type ViewMode = 'workspace' | 'clone' | 'kanban';

function App() {
    const [selectedWorkspace, setSelectedWorkspace] = useState<main.Workspace | null>(null);
    const [viewMode, setViewMode] = useState<ViewMode>('workspace');
    const [sidebarKey, setSidebarKey] = useState(0); // Used to force sidebar refresh

    const handleWorkspaceSelect = (workspace: main.Workspace) => {
        setSelectedWorkspace(workspace);
        setViewMode('workspace');
    };

    const handleNewWorkspace = () => {
        setViewMode('clone');
    };

    const handleRepositoryCloned = () => {
        // Refresh the sidebar to show the new workspace
        setSidebarKey(prev => prev + 1);
        setViewMode('workspace');
    };

    const handlePRDSaved = () => {
        // Refresh the sidebar to update PRD status
        setSidebarKey(prev => prev + 1);
    };

    const handleViewKanban = () => {
        setViewMode('kanban');
    };

    const handleBackToWorkspace = () => {
        setViewMode('workspace');
    };

    return (
        <div className="h-screen bg-background flex overflow-hidden">
            {/* Sidebar */}
            <WorkspaceSidebar
                key={sidebarKey}
                selectedWorkspace={selectedWorkspace?.name || null}
                onWorkspaceSelect={handleWorkspaceSelect}
                onNewWorkspace={handleNewWorkspace}
            />

            {/* Main Content */}
            <div className="flex-1 flex flex-col min-w-0">
                {/* Header */}
                <div className="border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                    <div className="p-4 sm:p-6">
                        <div className="flex items-center justify-between">
                            <div className="min-w-0 flex-1">
                                <h1 className="text-xl sm:text-2xl font-bold truncate">SpecPrint</h1>
                                <p className="text-muted-foreground text-sm sm:text-base hidden sm:block">
                                    AI-powered project management with Claude Code
                                </p>
                            </div>
                            
                            <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
                                {selectedWorkspace && viewMode === 'workspace' && (
                                    <Button 
                                        onClick={handleViewKanban}
                                        className="flex items-center gap-2"
                                    >
                                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 002 2m0 0v10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2h2a2 2 0 002-2" />
                                        </svg>
                                        <span className="hidden sm:inline">Kanban Board</span>
                                        <span className="sm:hidden">Kanban</span>
                                    </Button>
                                )}
                                
                                {(viewMode === 'clone' || viewMode === 'kanban') && (
                                    <Button 
                                        onClick={handleBackToWorkspace}
                                        variant="outline"
                                    >
                                        <span className="hidden sm:inline">← Back to Workspace</span>
                                        <span className="sm:hidden">← Back</span>
                                    </Button>
                                )}
                            </div>
                        </div>
                    </div>
                </div>

                {/* Main Content Area */}
                <div className="flex-1 overflow-auto min-h-0">
                    {viewMode === 'clone' ? (
                        <div className="p-4 sm:p-6 flex justify-center">
                            <div className="w-full max-w-2xl">
                                <Card className="mb-4">
                                    <CardHeader>
                                        <CardTitle>Clone New Repository</CardTitle>
                                        <CardDescription>
                                            Add a new repository to your workspaces
                                        </CardDescription>
                                    </CardHeader>
                                </Card>
                                <GitRepositoryCloner onRepositoryCloned={handleRepositoryCloned} />
                            </div>
                        </div>
                    ) : viewMode === 'kanban' ? (
                        <div className="p-4 sm:p-6">
                            <KanbanBoard selectedWorkspace={selectedWorkspace} />
                        </div>
                    ) : (
                        <div className="p-4 sm:p-6 flex justify-center">
                            <PRDInput
                                selectedWorkspace={selectedWorkspace}
                                onPRDSaved={handlePRDSaved}
                            />
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="border-t border-border p-3 sm:p-4">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between text-xs sm:text-sm text-muted-foreground gap-2 sm:gap-4">
                        <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-4 min-w-0">
                            {selectedWorkspace && (
                                <>
                                    <span className="truncate">Active: <strong>{selectedWorkspace.name}</strong></span>
                                    <span className="hidden sm:inline">•</span>
                                    <span className="whitespace-nowrap">PRD: {selectedWorkspace.hasPrd ? '✅ Available' : '⏳ Not Created'}</span>
                                </>
                            )}
                        </div>
                        <div className="whitespace-nowrap text-right">
                            {viewMode === 'workspace' ? 'Workspace View' : 
                             viewMode === 'kanban' ? 'Kanban Board' : 'Clone Repository'}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default App

