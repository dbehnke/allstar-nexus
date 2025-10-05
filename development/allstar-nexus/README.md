# Allstar Nexus

A full-stack application with a Next.js frontend and Go backend, delivered as a single binary.

## Directory Structure

*   `/frontend`: Contains the Next.js frontend application (TypeScript, Tailwind CSS)
*   `/backend`: Contains Go backend packages and utilities
*   `main.go`: Main application entry point that embeds and serves the frontend

## Getting Started

### Prerequisites

- Node.js and npm (for frontend development)
- Go 1.16+ (for backend development and building)

### Development

#### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

The frontend will be available at `http://localhost:3000`.

#### Building the Frontend

```bash
cd frontend
npm run build
```

This generates static files in `frontend/out/` which are embedded into the Go binary.

#### Running the Full Stack

```bash
# From the root directory
go run main.go
```

The server will start on `http://localhost:8080` and serve both the frontend and API.

#### Building for Production

```bash
# Build the frontend first
cd frontend && npm run build && cd ..

# Build the Go binary (includes embedded frontend)
go build -o allstar-nexus main.go

# Run the binary
./allstar-nexus
```

## Features

- ✅ Single binary deployment (frontend embedded in Go)
- ✅ Next.js with TypeScript and Tailwind CSS
- ✅ Go backend for API endpoints
- ✅ Hot reload for frontend development
- ✅ Static export for optimal performance

