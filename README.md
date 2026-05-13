# 🚀 FlyUp Backend

> **Empowering Student Entrepreneurs through Crowdfunding**

FlyUp is a modern crowdfunding platform specifically designed for student entrepreneurs. It connects innovative student projects ("Pioneers") with investors ("Boosters") through a milestone-based funding and execution system with transparent progress tracking and community voting.

[![Go Version](https://img.shields.io/badge/Go-1.25.3-00ADD8?style=flat&logo=go)](https://golang.org)
[![Fiber](https://img.shields.io/badge/Fiber-v3-00ACD7?style=flat)](https://gofiber.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io)

---

## 📋 Table of Contents

- [Features](#-features)
- [Technology Stack](#-technology-stack)
- [Architecture](#-architecture)
- [Getting Started](#-getting-started)
- [Environment Variables](#-environment-variables)
- [API Documentation](#-api-documentation)
- [Project Structure](#-project-structure)
- [Development](#-development)
- [Testing](#-testing)
- [Deployment](#-deployment)
- [Contributing](#-contributing)
- [License](#-license)

---

## ✨ Features

### Core Features

#### 🎓 **For Students (Pioneers)**
- Create and manage crowdfunding projects
- Multi-phase milestone tracking with submission system
- Project state management (draft → review → funding → executing → closed)
- Rich media support (images, videos)
- Project stories, FAQs, and update posting
- Meeting scheduling with investors
- Fund disbursement based on milestone completion

#### 💰 **For Investors (Boosters)**
- Browse and discover student projects
- Secure investment with Stripe PromptPay integration
- Vote on milestone completion (democratic approval)
- Track investment portfolio
- Receive profit distributions quarterly
- Request refunds during funding period
- Direct communication with pioneers

#### 🛡️ **For Administrators**
- Project approval/rejection workflow
- User verification management (student ID, national ID)
- Milestone submission review
- Refund request processing
- Disbursement confirmation
- Profit pool management
- University and domain management
- Complaint handling system

### Key Capabilities

- **Secure Authentication:** JWT + Google OAuth 2.0
- **Payment Processing:** Stripe API with PromptPay QR code
- **Milestone Voting:** Democratic decision-making with automatic tallying
- **Profit Sharing:** Quarterly profit distribution to investors
- **AI-Powered Chat:** Support system with intent recognition
- **Email Notifications:** Automated email updates via Resend
- **File Storage:** Cloudinary CDN integration
- **Real-time Updates:** Project status and funding tracking

---

## 🛠 Technology Stack

### Backend
- **Language:** Go 1.25.3
- **Web Framework:** Fiber v3 (high-performance HTTP framework)
- **ORM:** GORM (PostgreSQL)
- **Validation:** go-playground/validator

### Database & Cache
- **Primary Database:** PostgreSQL 18
- **Cache:** Redis 7
- **Migrations:** Auto-migration via GORM

### External Services
- **Payment:** Stripe API v85 (PromptPay integration)
- **Email:** Resend Email API
- **Storage:** Cloudinary CDN
- **AI/LLM:** OpenAI API (chat support)
- **OAuth:** Google OAuth 2.0

### DevOps
- **Containerization:** Docker & Docker Compose
- **API Docs:** Swagger/OpenAPI (swaggo)
- **Testing:** testify framework

---

## 🏗 Architecture

### Clean Architecture Pattern

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Layer (Fiber)                    │
│                   Handlers & Routes                      │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                   Service Layer                          │
│              Business Logic & Orchestration              │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                 Repository Layer                         │
│              Database Access & Queries                   │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                  Domain Models                           │
│              Core Business Entities                      │
└──────────────────────────────────────────────────────────┘
```

### Key Design Principles
- **Separation of Concerns:** Clear boundaries between layers
- **Dependency Injection:** Services injected via constructors
- **Interface-Based Design:** Easy mocking and testing
- **Repository Pattern:** Database abstraction
- **DTO Pattern:** Request/response data transformation

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.25.3+** - [Install Go](https://golang.org/dl/)
- **Docker & Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)
- **PostgreSQL 18** (or use Docker)
- **Redis 7** (or use Docker)
- **Stripe Account** - [Sign up for Stripe](https://stripe.com)
- **Cloudinary Account** - [Sign up for Cloudinary](https://cloudinary.com)
- **Resend Account** - [Sign up for Resend](https://resend.com)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/your-org/flyup-backend.git
cd flyup-backend
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Start Docker services**
```bash
docker-compose up -d
```

5. **Run migrations** (automatic on startup)
```bash
go run main.go
```

6. **Access the application**
- API Server: `http://localhost:8080`
- Swagger Docs: `http://localhost:8080/swagger/index.html`

### Quick Start with Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

---

## 🔐 Environment Variables

Create a `.env` file in the root directory:

```env
# Server Configuration
PORT=8080
FRONTEND_URL=http://localhost:3000

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=flyup
DB_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT Authentication
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_REFRESH_SECRET=your-super-secret-refresh-key-change-this-too

# Google OAuth
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# Stripe Payment
STRIPE_SECRET_KEY=sk_test_xxxxxxxxxxxxxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxxxxx

# Cloudinary CDN
CLOUDINARY_CLOUD_NAME=your-cloud-name
CLOUDINARY_API_KEY=your-api-key
CLOUDINARY_API_SECRET=your-api-secret

# Email Service (Resend)
RESEND_API_KEY=re_xxxxxxxxxxxx

# OpenAI (for chat support)
OPENAI_API_KEY=sk-xxxxxxxxxxxx
```

---

## 📚 API Documentation

### Access Swagger UI

Once the server is running, visit:
```
http://localhost:8080/swagger/index.html
```

### Generate Swagger Docs

After modifying API endpoints:
```bash
swag init
```

### API Endpoint Categories

| Category | Base Path | Description |
|----------|-----------|-------------|
| Authentication | `/auth/*` | Login, signup, OAuth |
| User Profile | `/user/*` | Profile management, verification |
| Projects (Public) | `/projects/*` | Browse projects |
| Pioneer | `/pioneer/*` | Project creation & management |
| Booster | `/booster/*` | Investment & voting |
| Admin | `/admin/*` | Administration panel |
| Investments | `/investments/*` | Investment operations |
| Chat | `/chat/*` | Support chat system |

### Key Endpoints Examples

**Authentication:**
```http
POST /signup                 # Register new user
POST /signin                 # Login
GET  /auth/google            # Google OAuth
POST /auth/refresh           # Refresh token
```

**Projects:**
```http
GET  /projects               # List public projects
GET  /projects/:id           # Project details
POST /pioneer/projects       # Create project
PATCH /pioneer/projects/:id  # Update project
```

**Investments:**
```http
POST /investments            # Create investment
GET  /investments            # My investments
POST /investments/milestones/:id/vote  # Vote on milestone
```

**Admin:**
```http
PATCH /admin/projects/:id/approve      # Approve project
GET   /admin/investments/refund-requests  # View refunds
PATCH /admin/disbursements/:id/confirm    # Confirm payout
```

---

## 📁 Project Structure

```
flyup-backend/
├── main.go                          # Application entry point
├── config/
│   └── appConfig.go                # Configuration & database setup
├── internal/
│   ├── api/
│   │   └── rest/
│   │       ├── server.go           # Server initialization
│   │       ├── middlewares.go      # Auth & security middleware
│   │       ├── response.go         # Response helpers
│   │       └── handlers/           # HTTP handlers (13 files)
│   ├── domain/                     # Domain models (entities)
│   │   ├── user.go
│   │   ├── project.go
│   │   ├── investment.go
│   │   ├── milestone.go
│   │   └── ...
│   ├── dto/                        # Data Transfer Objects
│   │   ├── authDTO.go
│   │   ├── projectDTO.go
│   │   └── ...
│   ├── service/                    # Business logic layer
│   │   ├── userService.go
│   │   ├── projectService.go
│   │   ├── investmentService.go
│   │   └── ...
│   ├── repository/                 # Data access layer
│   │   ├── userRepository.go
│   │   ├── projectRepository.go
│   │   └── ...
│   └── helper/                     # Utility functions
│       ├── auth.go                 # JWT & password hashing
│       ├── cloudinary.go           # Image upload
│       ├── oauth.go                # Google OAuth
│       └── utility.go              # Helpers
├── pkg/
│   ├── redis/                      # Redis client wrapper
│   ├── notification/               # Email service
│   └── llm/                        # AI chat integration
├── docs/                           # Swagger generated docs
├── docker-compose.yml              # Docker services
├── Dockerfile                      # Production build
├── go.mod                          # Go dependencies
└── .env.example                    # Environment template
```

---

## 💻 Development

### Running the Server

**Development mode:**
```bash
go run main.go
```

**With auto-reload (using Air):**
```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

**Build for production:**
```bash
go build -o flyup-backend main.go
./flyup-backend
```

### Database Migrations

GORM auto-migrates on startup. To manually trigger:

```go
// In config/appConfig.go
db.AutoMigrate(
    &domain.User{},
    &domain.Project{},
    &domain.Investment{},
    // ... other models
)
```

### Code Generation

**Generate Swagger docs:**
```bash
swag init
```

**Generate mocks for testing:**
```bash
mockgen -source=internal/repository/userRepository.go -destination=internal/repository/mock_user_repository.go
```

### Code Style

Follow Go best practices:
```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run

# Vet code
go vet ./...
```

---

## 🧪 Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Package Tests

```bash
go test ./internal/service/...
```

### Test Structure

Tests are located next to their implementation files:
```
internal/service/
├── userService.go
├── userService_test.go
├── projectService.go
└── projectService_test.go
```

---

## 🚢 Deployment

### Docker Production Build

```bash
# Build image
docker build -t flyup-backend:latest .

# Run container
docker run -p 8080:8080 --env-file .env flyup-backend:latest
```

### Environment-Specific Configuration

**Production checklist:**
- [ ] Change all secret keys
- [ ] Enable HTTPS/TLS
- [ ] Configure CORS properly
- [ ] Set up database backups
- [ ] Configure Redis persistence
- [ ] Enable rate limiting
- [ ] Set up monitoring/logging
- [ ] Configure Stripe production keys
- [ ] Update frontend URL

### Docker Compose Production

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
```

---

## 🔒 Security Features

- **JWT Authentication** with encrypted user IDs
- **bcrypt Password Hashing** (cost factor: 10)
- **CORS Protection** with whitelist
- **XSS Protection Headers**
- **CSRF Middleware**
- **SQL Injection Prevention** via parameterized queries (GORM)
- **Rate Limiting** (TODO: implement)
- **Stripe Webhook Signature Verification**
- **User Suspension System**
- **ID & Student Card Verification**

---

## 📊 Key Business Logic

### Project Lifecycle

```
draft → pending_review → funding → executing → closed
  ↓                         ↓          ↓
cancelled              cancelled   suspended
```

### Milestone Workflow

```
created → submitted → voting → approved/rejected
                                    ↓
                                  paid
                                    ↓
                           disbursement created
```

### Investment Flow

```
create investment → stripe payment → payment succeeded
        ↓                                   ↓
transaction pending                investment verified
        ↓                                   ↓
   (expires after 15min)            project funded++
```

### Platform Fees

- **Platform Fee:** 3% of investment amount
- **VAT:** 7% on platform fee
- **Stripe Fee:** Varies (deducted by Stripe)
- **Max Investment per Transaction:** 500,000 THB

---

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

1. **Fork the repository**
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Commit your changes** (`git commit -m 'Add amazing feature'`)
4. **Push to the branch** (`git push origin feature/amazing-feature`)
5. **Open a Pull Request**

### Commit Message Convention

```
feat: add new feature
fix: bug fix
docs: documentation update
style: code style changes
refactor: code refactoring
test: add tests
chore: maintenance tasks
```

---

## 📝 Known Issues & TODOs

### High Priority
- [ ] Implement rate limiting on API endpoints
- [ ] Add comprehensive input sanitization (XSS)
- [ ] Set up automated testing pipeline
- [ ] Implement WebSocket for real-time notifications
- [ ] Add email notification sending integration
- [ ] Improve error handling and logging

### Medium Priority
- [ ] Add full-text search for projects
- [ ] Optimize database queries with proper indexes
- [ ] Implement Redis caching strategy
- [ ] Add pagination to all list endpoints
- [ ] Create admin dashboard analytics

### Low Priority
- [ ] Add project recommendation algorithm improvements
- [ ] Implement advanced filtering options
- [ ] Add export functionality for reports
- [ ] Create backup automation scripts

---

## 📞 Support

For questions or issues:
- **GitHub Issues:** [Create an issue](https://github.com/your-org/flyup-backend/issues)
- **Email:** support@flyup.app
- **Documentation:** See `agent.md` for detailed technical docs

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Fiber](https://gofiber.io) - Fast HTTP framework
- [GORM](https://gorm.io) - Fantastic ORM library
- [Stripe](https://stripe.com) - Payment processing
- [Cloudinary](https://cloudinary.com) - Media management
- [Resend](https://resend.com) - Email service

---

<p align="center">
  Made with ❤️ by the FlyUp Team
</p>

<p align="center">
  <a href="https://flyup.app">Website</a> •
  <a href="https://docs.flyup.app">Documentation</a> •
  <a href="https://github.com/your-org/flyup-backend">GitHub</a>
</p>
