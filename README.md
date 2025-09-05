# Indico Test Assignment

A full-stack application demonstrating modern software engineering practices with a high-performance Go backend and React TypeScript frontend.

## 🏗️ Project Structure

```
indico-test/
├── backend/                    # Go backend service
│   ├── cmd/                   # Application entry points
│   ├── internal/              # Private application code
│   ├── test/                  # Integration tests
│   ├── monitoring/            # Prometheus & Grafana configs
│   └── docker-compose.yml     # Backend services orchestration
├── frontend/                   # React TypeScript frontend
│   ├── src/                   # Source code
│   ├── public/                # Static assets
│   └── dist/                  # Built application
└── README.md                  # This file
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Node.js 18+ (for frontend development)
- Go 1.21+ (for backend development)

### Running the Complete Application

1. **Start the backend services**:

   ```bash
   cd backend
   cp .env.example .env
   docker-compose up -d
   ```

2. **Start the frontend**:

   ```bash
   cd frontend
   cp .env.example .env
   npm install
   npm run dev
   ```

3. **Access the applications**:
   - Frontend: http://localhost:5173 (User Management Interface)
   - Backend API: http://localhost:8080 (Order Management API)
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090

> **Note**: The frontend and backend are separate applications. The frontend demonstrates modern React patterns with JSONPlaceholder API, while the backend showcases Go microservice architecture for order processing.

## 🔧 Backend Features

- **Order Management**: High-performance order processing with stock management
- **Concurrency Safe**: Handles 500+ concurrent orders without overselling
- **Background Jobs**: Settlement processing with cancellable worker pools
- **Monitoring**: Prometheus metrics and Grafana dashboards
- **Clean Architecture**: Layered design with dependency injection
- **Database**: PostgreSQL with optimistic locking and migrations

### Tech Stack

- Go 1.21+ with Gin framework
- PostgreSQL for data persistence
- Docker for containerization
- Prometheus & Grafana for monitoring

## 🎨 Frontend Features

- **User Management**: Add, delete, and search users with company information
- **Real-time Search**: Debounced search with instant filtering by name
- **Data Table**: Paginated user list displaying ID, name, email, and company
- **CRUD Operations**: Create new users and delete existing ones
- **Toast Notifications**: Success/error feedback for all user actions
- **Responsive Design**: Material-UI components optimized for all devices
- **Loading States**: Smooth UX with loading indicators and error handling

### Tech Stack

- React 19 with TypeScript
- Material-UI (MUI) for component library
- React Query (@tanstack/react-query) for server state management
- Vite for build tooling and development
- JSONPlaceholder API for demo data

## 🧪 Testing

### Backend Tests

```bash
cd backend
go test ./... -v
```

### Frontend Tests

```bash
cd frontend
npm run lint
npm run build
```

## 📊 Monitoring & Observability

The backend includes comprehensive monitoring:

- **Health Checks**: `/health` endpoint for service status
- **Metrics**: Prometheus metrics at `/metrics`
- **Dashboards**: Pre-configured Grafana dashboards
- **Logging**: Structured JSON logging

## 🚀 Production Deployment

### Environment Configuration

Both backend and frontend use environment variables for configuration:

- Backend: See `backend/.env.example`
- Frontend: See `frontend/.env.example`

### Docker Deployment

```bash
# Backend services
cd backend
docker-compose up -d

# Frontend (build for production)
cd frontend
npm run build
# Serve dist/ directory with your preferred web server
```

## 📝 Development

### Backend Development

```bash
cd backend
go mod download
go run cmd/server/main.go
```

### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

## 🔒 Security Considerations

- Environment variables for sensitive configuration
- CORS properly configured
- Input validation on all endpoints
- Database connection pooling and prepared statements
- Graceful shutdown handling

## 📚 Documentation

- [Backend README](./backend/README.md) - Detailed backend documentation
- [Frontend README](./frontend/README.md) - Detailed frontend documentation
- [Environment Configuration](./backend/ENV_CONFIGURATION.md) - Environment setup guide

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## 📄 License

This project is part of a technical assessment for Indico.

---

**Assignment Status**: ✅ Complete with both backend and frontend implementations, monitoring, and proper documentation.
