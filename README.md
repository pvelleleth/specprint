# SpecPrint

**AI-powered project management with Claude Code**

SpecPrint is a desktop application that transforms Product Requirements Documents (PRDs) into structured, actionable project tasks using AI. It features an interactive Kanban board for task management and Git repository integration for seamless project workflow.

## ✨ Features

- **AI-Powered Task Generation**: Convert PRDs into structured Epics → Stories → Tasks hierarchy
- **Interactive Kanban Board**: Drag-and-drop task management with real-time updates
- **Git Repository Integration**: Clone and manage repositories directly in the app
- **Workspace Management**: Organize projects with dedicated workspaces
- **PRD Editor**: Create and edit Product Requirements Documents with live preview
- **Task Dependencies**: Manage task relationships and priorities
- **Cross-Platform**: Built with Wails (Go + React) for Windows, macOS, and Linux

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Git

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

3. **Set up environment variables**
   Create a `.env` file in the project root:
   ```env
   OPENAI_API_KEY=your_openai_api_key_here
   ```

4. **Run in development mode**
   ```bash
   wails dev
   ```

### Building for Production

```bash
wails build
```

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
- The system will create a structured hierarchy:
  - **Epics**: High-level features (3-6 epics)
  - **Stories**: User stories within each epic (2-5 stories per epic)
  - **Tasks**: Implementation tasks (3-8 tasks per story)

### 4. Manage with Kanban
- Click "Kanban Board" to view your tasks
- Drag and drop tasks between columns (To Do, In Progress, Done)
- Update task status and track progress
- All changes are automatically saved

## 🏗️ Architecture

- **Backend**: Go with Wails framework
- **Frontend**: React + TypeScript + Tailwind CSS
- **AI**: OpenAI GPT-4 integration for task generation
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

## 🔧 Configuration

### Environment Variables
- `OPENAI_API_KEY`: Required for AI task generation

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
- Check the [Task Generation and Kanban Guide](TASK_GENERATION_AND_KANBAN_GUIDE.md)
- Review the [Kanban Enhancements](KANBAN_ENHANCEMENTS.md) for latest features
- Open an issue on GitHub

---

**Built with ❤️ using Wails, React, OpenAI, and Claude**
