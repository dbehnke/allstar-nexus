# Allstar Nexus Project

This document outlines the development plan and architecture for the Allstar Nexus project.

## Architecture

Allstar Nexus is a full-stack application that combines:
- **Frontend**: Next.js with TypeScript and Tailwind CSS (static export)
- **Backend**: Go web server that embeds the frontend and serves API endpoints
- **Deployment**: Single binary deployment (frontend assets embedded in Go binary)

## Phased Plan

### Phase 1: Project Setup ✅ COMPLETED

1.  ✅ **Created `copilot-instructions.md`**: Project planning and instructions
2.  ✅ **Initialized Git and GitHub Repository**: Repository created at `github.com/dbehnke/allstar-nexus`
3.  ✅ **Scaffolded the Project**: 
    - Next.js application with TypeScript, ESLint, and Tailwind CSS
    - Go backend with embedded frontend serving
    - Monorepo structure with separate frontend/backend directories

### Phase 2: Core Feature Development (NEXT)

Goals for this phase:
1.  **API Endpoints**: Set up RESTful API structure in Go
2.  **Database Integration**: Connect to database (PostgreSQL/SQLite)
3.  **Authentication**: Implement user authentication system
4.  **Frontend Components**: Build reusable UI components
5.  **State Management**: Set up frontend state management

### Phase 3: Advanced Features

1.  **Real-time Communication**: WebSocket support
2.  **File Uploads**: Handle file uploads and storage
3.  **Background Jobs**: Implement job queue system
4.  **Caching**: Add Redis or in-memory caching

### Phase 4: Deployment and DevOps

1.  **CI/CD Pipeline**: GitHub Actions for automated testing and deployment
2.  **Docker**: Containerization setup
3.  **Monitoring**: Application monitoring and logging
4.  **Documentation**: API documentation and user guides

## Current Project Structure

```
allstar-nexus/
├── main.go                  # Main application entry point
├── go.mod                   # Go module definition
├── frontend/                # Next.js application
│   ├── src/
│   │   └── app/            # Next.js App Router pages
│   ├── public/             # Static assets
│   ├── package.json
│   └── next.config.ts      # Next.js config (static export enabled)
└── backend/                # Go backend packages
    └── server/             # Server utilities
        └── frontend.go     # Frontend serving utilities
```

## Development Workflow

1. **Frontend Development**: Run `npm run dev` in `frontend/` directory
2. **Build Frontend**: Run `npm run build` in `frontend/` directory
3. **Run Full Stack**: Run `go run main.go` from root directory
4. **Build for Production**: Build frontend, then `go build -o allstar-nexus main.go`

