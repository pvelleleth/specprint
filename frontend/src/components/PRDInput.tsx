import { useState, useRef } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { SaveWorkspacePRD } from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";

interface PRDResult {
  success: boolean;
  message: string;
  path?: string;
}

interface PRDInputProps {
  selectedWorkspace: main.Workspace | null;
  onPRDSaved?: () => void;
}

export function PRDInput({ selectedWorkspace, onPRDSaved }: PRDInputProps) {
  const [prdContent, setPrdContent] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [result, setResult] = useState<PRDResult | null>(null);
  const [inputMethod, setInputMethod] = useState<'text' | 'file'>('text');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleSavePRD = async () => {
    if (!selectedWorkspace) {
      setResult({
        success: false,
        message: 'Please select a workspace first'
      });
      return;
    }

    if (!prdContent.trim()) {
      setResult({
        success: false,
        message: 'Please enter PRD content or upload a file'
      });
      return;
    }

    setIsSaving(true);
    setResult(null);

    try {
      const saveResult = await SaveWorkspacePRD(selectedWorkspace.name, prdContent.trim());
      setResult(saveResult);
      if (saveResult.success && onPRDSaved) {
        onPRDSaved();
      }
    } catch (error) {
      setResult({
        success: false,
        message: `Error: ${error instanceof Error ? error.message : 'Unknown error occurred'}`
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setPrdContent(e.target.value);
    // Clear previous result when user starts typing
    if (result) {
      setResult(null);
    }
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file size (10MB limit)
    const maxSize = 10 * 1024 * 1024; // 10MB
    if (file.size > maxSize) {
      setResult({
        success: false,
        message: 'File size exceeds 10MB limit. Please choose a smaller file.'
      });
      return;
    }

    // Validate file type
    const allowedTypes = ['.txt', '.md'];
    const fileExtension = file.name.toLowerCase().substring(file.name.lastIndexOf('.'));
    if (!allowedTypes.includes(fileExtension)) {
      setResult({
        success: false,
        message: 'Invalid file type. Please upload a .txt or .md file.'
      });
      return;
    }

    const reader = new FileReader();
    reader.onload = (event) => {
      const content = event.target?.result as string;
      setPrdContent(content);
      setResult(null);
    };
    reader.readAsText(file);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.ctrlKey && !isSaving && selectedWorkspace) {
      handleSavePRD();
    }
  };

  const clearContent = () => {
    setPrdContent('');
    setResult(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  if (!selectedWorkspace) {
    return (
      <Card className="w-full max-w-4xl">
        <CardHeader>
          <CardTitle>Product Requirements Document (PRD)</CardTitle>
          <CardDescription>
            Please select a workspace from the sidebar to create or edit a PRD.
          </CardDescription>
        </CardHeader>
        <CardContent className="text-center py-12">
          <div className="max-w-md mx-auto">
            <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <h3 className="text-lg font-medium mb-2">No Workspace Selected</h3>
            <p className="text-muted-foreground mb-4">
              Select a workspace from the sidebar or clone a new repository to get started.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="w-full max-w-4xl">
      <CardHeader>
        <CardTitle>Product Requirements Document (PRD)</CardTitle>
        <CardDescription className="flex items-center justify-between">
          <span>Create or edit a PRD for your workspace</span>
          <div className="flex items-center gap-2 text-sm bg-muted px-3 py-1 rounded-full">
            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
            <span className="font-medium">{selectedWorkspace.name}</span>
          </div>
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Workspace Info */}
        {selectedWorkspace.hasPrd && (
          <div className="bg-green-50 border border-green-200 rounded-lg p-3">
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-green-500 rounded-full flex items-center justify-center">
                <svg className="w-2 h-2 text-white" fill="currentColor" viewBox="0 0 8 8">
                  <path d="M6.564.75l-3.59 3.612-1.538-1.55L0 4.26 2.974 7.25 8 2.193z"/>
                </svg>
              </div>
              <p className="text-sm text-green-800 font-medium">
                This workspace already has a PRD. Saving will overwrite the existing file.
              </p>
            </div>
          </div>
        )}

        {/* Input Method Toggle */}
        <div className="flex space-x-2">
          <Button
            variant={inputMethod === 'text' ? 'default' : 'outline'}
            onClick={() => setInputMethod('text')}
            size="sm"
          >
            Paste Text
          </Button>
          <Button
            variant={inputMethod === 'file' ? 'default' : 'outline'}
            onClick={() => setInputMethod('file')}
            size="sm"
          >
            Upload File
          </Button>
        </div>

        {/* Text Input */}
        {inputMethod === 'text' && (
          <div className="space-y-2">
            <label htmlFor="prd-content" className="text-sm font-medium">
              PRD Content
            </label>
            <textarea
              id="prd-content"
              value={prdContent}
              onChange={handleContentChange}
              onKeyPress={handleKeyPress}
              placeholder="Paste your PRD content here...&#10;&#10;Example:&#10;# Project Overview&#10;This project aims to create a web application that...&#10;&#10;## Features&#10;- User authentication&#10;- Data visualization&#10;- API integration"
              className="w-full h-64 px-3 py-2 border border-input bg-background rounded-md focus:outline-none focus:ring-2 focus:ring-ring resize-none font-mono text-sm"
              disabled={isSaving}
            />
            <p className="text-xs text-muted-foreground">
              Press Ctrl+Enter to save. Supports Markdown formatting.
            </p>
          </div>
        )}

        {/* File Upload */}
        {inputMethod === 'file' && (
          <div className="space-y-2">
            <label htmlFor="prd-file" className="text-sm font-medium">
              Upload PRD File
            </label>
            <input
              id="prd-file"
              ref={fileInputRef}
              type="file"
              accept=".txt,.md"
              onChange={handleFileUpload}
              className="w-full px-3 py-2 border border-input bg-background rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
              disabled={isSaving}
            />
            <p className="text-xs text-muted-foreground">
              Supported formats: .txt, .md (max 10MB)
            </p>
            
            {/* Show uploaded content preview */}
            {prdContent && (
              <div className="space-y-2">
                <label className="text-sm font-medium">File Content Preview</label>
                <textarea
                  value={prdContent}
                  onChange={handleContentChange}
                  onKeyPress={handleKeyPress}
                  className="w-full h-32 px-3 py-2 border border-input bg-background rounded-md focus:outline-none focus:ring-2 focus:ring-ring resize-none font-mono text-sm"
                  disabled={isSaving}
                />
              </div>
            )}
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex space-x-2">
          <Button 
            onClick={handleSavePRD} 
            disabled={isSaving || !prdContent.trim() || !selectedWorkspace}
            className="flex-1"
          >
            {isSaving ? (
              <div className="flex items-center gap-2">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                Saving PRD to {selectedWorkspace.name}...
              </div>
            ) : (
              `Save PRD to ${selectedWorkspace.name}`
            )}
          </Button>
          
          <Button 
            onClick={clearContent}
            variant="outline"
            disabled={isSaving || !prdContent.trim()}
          >
            Clear
          </Button>
        </div>

        {/* Character Count */}
        <div className="text-xs text-muted-foreground text-right">
          {prdContent.length} characters
        </div>

        {/* Result Display */}
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
                    PRD saved to: {result.path}
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