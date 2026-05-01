# 💬 GoChat — Production Real-time Chat App

A production-grade real-time chat application built with Go WebSockets, Next.js 14, and TypeScript. Features multiple chat rooms, user authentication, and message history.

## Features
- ⚡ Real-time messaging with native Go WebSockets
- 🏠 Multiple chat rooms (General, Tech, Gaming, Music, Random)
- 🔐 JWT authentication (register/login)
- 📜 Message history loaded from MongoDB on join
- 👥 Live online users list per room
- 🔔 Join/leave notifications
- 🟢 Connection status indicator
- 🎨 Clean UI with Next.js + shadcn/ui

## Tech Stack
| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 14 + TypeScript |
| Styling | Tailwind CSS + shadcn/ui |
| Backend | Go (Golang) |
| WebSockets | Gorilla WebSocket |
| Database | MongoDB |
| Auth | JWT (golang-jwt) |
| Password | bcrypt |

## Why Go for WebSockets?
Go's goroutines make it extremely efficient for handling thousands of concurrent WebSocket connections with minimal memory usage — far more scalable than Node.js for real-time applications.

## Getting Started

### Backend (Go)
```bash
# Install dependencies
go mod tidy

# Run the server
go run main.go
```

Server runs on `http://localhost:8080`

### Frontend (Next.js)
```bash
cd frontend
npm install
npm run dev
```

Frontend runs on `http://localhost:3000`

### Environment Variables
Create a `.env` file in the root:
PORT=8080
MONGO_URI=mongodb://localhost:27017
JWT_SECRET=your_secret_key
## Project Structure
go-chat/
├── config/
│   └── config.go          # Environment configuration
├── handlers/
│   ├── auth.go            # Register/Login handlers
│   └── websocket.go       # WebSocket hub and client
├── middleware/
│   └── auth.go            # JWT middleware
├── models/
│   ├── user.go            # User model
│   └── message.go         # Message model
├── frontend/
│   ├── app/
│   │   ├── login/         # Login page
│   │   ├── register/      # Register page
│   │   └── chat/          # Chat page
│   ├── components/
│   │   ├── Navbar.tsx
│   │   ├── MessageBubble.tsx
│   │   └── UsersList.tsx
│   ├── context/
│   │   └── AuthContext.tsx
│   └── lib/
│       ├── api.ts
│       └── types.ts
├── main.go                # Entry point
├── go.mod
└── go.sum

## How It Works
- Client connects via WebSocket with JWT token as query param
- Go hub manages all connected clients in memory using goroutines
- Messages are broadcast to all clients in the same room instantly
- Message history is stored in MongoDB and loaded on room join
- Each room maintains its own set of connected clients
