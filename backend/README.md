# Backend for Allstar Nexus

This directory contains Go packages and utilities for the Allstar Nexus backend.

## Structure

- `server/` - Web server utilities and handlers

## Running the Application

The main application is located at the root of the repository. To run it:

```bash
cd /home/dbehnke/development/allstar-nexus
go run main.go
```

Or build it:

```bash
go build -o allstar-nexus main.go
./allstar-nexus
```

The server will start on port 8080 and serve the Next.js frontend along with API endpoints.

