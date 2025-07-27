# SpecPrint

**AI-powered project management with Claude Code**

SpecPrint is a desktop application that transforms Product Requirements Documents (PRDs) into structured, actionable project tasks using AI. It features an interactive Kanban board for task management and Git repository integration for seamless project workflow.

## ✨ Features

- **AI-Powered Task Generation**: Convert PRDs into structured tasks using Anthropic's Claude.
- **Interactive Kanban Board**: Drag-and-drop task management with real-time updates
- **Git Repository Integration**: Clone and manage repositories directly in the app
- **Workspace Management**: Organize projects with dedicated workspaces
- **PRD Editor**: Create and edit Product Requirements Documents with live preview
- **Task Dependencies**: Manage task relationships and priorities
- **Cross-Platform**: Built with Wails (Go + React) for Windows, macOS, and Linux (built for macOS first)

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Git
- [Claude Code CLI](https://www.npmjs.com/package/@anthropic-ai/claude-code)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd specprint
   ```

2. **Install dependencies**
   ```bash
   # Install Go dependencies
   go mod tidy
   
   # Install frontend dependencies
   cd frontend
   npm install
   cd ..
   ```

3. **Install and set up Claude Code**
   This application uses the `claude-code` CLI to interact with Anthropic's AI.
   ```bash
   # Install the CLI globally
   npm install -g @anthropic-ai/claude-code

   # Follow the prompts to log in to your Anthropic account
   claude login
   ```
   
   Ensure the `claude` command is available in your system's PATH.

4. **Run in development mode**
   ```bash
   wails dev
   ```

### Building for Production

```bash
wails build
```

### macOS Installation Note

The pre-built application for macOS is not yet notarized by Apple. To run it for the first time, you may need to:
1.  Right-click the application icon.
2.  Select "Open".
3.  A dialog will appear. Click "Open" again to confirm you want to run the application.

You should only need to do this the first time you launch the app.

## 📖 How to Use

### 1. Create a Workspace
- Click "New Workspace" in the sidebar
- Enter a Git repository URL or create a local workspace
- The repository will be cloned to `~/.aicodingtool/repos/[workspace-name]`

### 2. Write a PRD
- Select your workspace from the sidebar
- Use the PRD editor to write your Product Requirements Document
- Include project overview, features, and technical requirements
- Save the PRD to your workspace

### 3. Generate Tasks
- Click "Generate from PRD" to process your document with AI

### 4. Manage with Kanban
- Click "Kanban Board" to view your tasks
- Drag and drop tasks between columns (To Do, In Progress, Done)
- Update task status and track progress
- All changes are automatically saved

## 🏗️ Architecture

- **Backend**: Go with Wails framework
- **Frontend**: React + TypeScript + Tailwind CSS
- **AI**: Anthropic Claude integration via the `@anthropic-ai/claude-code` CLI.
- **Storage**: Local file system with JSON persistence
- **Git**: go-git for repository management

## 📁 Project Structure

```
specprint/
├── app.go                 # Main application logic
├── main.go               # Wails entry point
├── frontend/             # React frontend
│   ├── src/
│   │   ├── components/   # React components
│   │   │   ├── Kanban/   # Kanban board components
│   │   │   └── ui/       # UI components
│   │   └── App.tsx       # Main app component
│   └── package.json
├── pkg/                  # Go packages
│   ├── claude/           # AI integration
│   └── git/              # Git utilities
└── wails.json           # Wails configuration
```

## �� Configuration

### Claude Code CLI
This application relies on your local installation and configuration of the `@anthropic-ai/claude-code` CLI tool. There are no separate API keys to manage within this project. All AI interaction is handled through the CLI, using the account you logged into via `claude login`.

### Workspace Storage
Workspaces are stored in `~/.aicodingtool/repos/` with the following structure:
```
~/.aicodingtool/repos/
├── workspace-1/
│   ├── PRD.md           # Product Requirements Document
│   └── [cloned-repo]/   # Git repository contents
└── workspace-2/
    └── ...
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

**Important**: This license prevents commercialization of the software. Any derivative works must also be open source and cannot be sold or used commercially without permission.

## 🆘 Support

For issues and questions:

- Open an issue on GitHub

---

**Built with ❤️ using Wails, React, and Anthropic Claude**
