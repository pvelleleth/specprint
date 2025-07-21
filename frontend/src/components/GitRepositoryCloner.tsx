import { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { CloneRepository } from "../../wailsjs/go/main/App";

interface CloneResult {
  success: boolean;
  message: string;
  path?: string;
}

interface GitRepositoryClonerProps {
  onRepositoryCloned?: () => void;
}

export function GitRepositoryCloner({ onRepositoryCloned }: GitRepositoryClonerProps) {
  const [repoURL, setRepoURL] = useState('');
  const [isCloning, setIsCloning] = useState(false);
  const [result, setResult] = useState<CloneResult | null>(null);

  const handleClone = async () => {
    if (!repoURL.trim()) {
      setResult({
        success: false,
        message: 'Please enter a repository URL'
      });
      return;
    }

    setIsCloning(true);
    setResult(null);

    try {
      const cloneResult = await CloneRepository(repoURL.trim());
      setResult(cloneResult);
      
      // Notify parent component if cloning was successful
      if (cloneResult.success && onRepositoryCloned) {
        onRepositoryCloned();
      }
    } catch (error) {
      setResult({
        success: false,
        message: `Error: ${error instanceof Error ? error.message : 'Unknown error occurred'}`
      });
    } finally {
      setIsCloning(false);
    }
  };

  const handleURLChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setRepoURL(e.target.value);
    // Clear previous result when user starts typing
    if (result) {
      setResult(null);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !isCloning) {
      handleClone();
    }
  };

  return (
    <Card className="w-full max-w-2xl">
      <CardHeader>
        <CardTitle>Clone Git Repository</CardTitle>
        <CardDescription>
          Clone a Git repository into the application's dedicated directory
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <label htmlFor="repo-url" className="text-sm font-medium">
            Repository URL
          </label>
          <input
            id="repo-url"
            type="text"
            value={repoURL}
            onChange={handleURLChange}
            onKeyPress={handleKeyPress}
            placeholder="https://github.com/user/repo.git or git@github.com:user/repo.git"
            className="w-full px-3 py-2 border border-input bg-background rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
            disabled={isCloning}
          />
          <p className="text-xs text-muted-foreground">
            Supports HTTPS and SSH URLs
          </p>
        </div>

        <Button 
          onClick={handleClone} 
          disabled={isCloning || !repoURL.trim()}
          className="w-full"
        >
          {isCloning ? (
            <div className="flex items-center gap-2">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              Cloning...
            </div>
          ) : (
            'Clone Repository'
          )}
        </Button>

        {result && (
          <div className={`p-4 rounded-lg border ${
            result.success 
              ? 'bg-green-50 border-green-200 text-green-800' 
              : 'bg-red-50 border-red-200 text-red-800'
          }`}>
            <div className="flex items-start gap-2">
              <div className={`w-5 h-5 rounded-full flex items-center justify-center ${
                result.success ? 'bg-green-500' : 'bg-red-500'
              }`}>
                <span className="text-white text-xs">
                  {result.success ? '✓' : '✗'}
                </span>
              </div>
              <div className="flex-1">
                <p className="font-medium">
                  {result.success ? 'Success!' : 'Error'}
                </p>
                <p className="text-sm mt-1">{result.message}</p>
                {result.success && result.path && (
                  <p className="text-xs mt-2 text-green-600">
                    Repository cloned to: {result.path}
                  </p>
                )}
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
} 