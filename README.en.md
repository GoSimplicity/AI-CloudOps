# AI-CloudOps

> **English | [中文](README.md)**

An AI-powered cloud-native operations management platform that integrates Kubernetes management, monitoring & alerting, and intelligent analysis.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.0+-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.25+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Stars](https://img.shields.io/github/stars/GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/stargazers)

![AI-CloudOps](https://socialify.git.ci/GoSimplicity/AI-CloudOps/image?description=1&font=Inter&forks=1&issues=1&name=1&pattern=Solid&pulls=1&stargazers=1&theme=Dark)

## Introduction

AI-CloudOps is a modern cloud-native operations management platform designed to improve operational efficiency and system stability through artificial intelligence technology.

### Core Features

- **Intelligent Operations**: Machine learning-based anomaly detection, fault prediction, and automated recovery
- **Kubernetes Management**: Complete cluster management, application deployment, and monitoring capabilities
- **Monitoring & Alerting**: Comprehensive monitoring and intelligent alerting based on Prometheus
- **Permission Management**: Multi-tenant RBAC permission control system
- **Ticketing System**: Complete operational ticketing workflow management
- **Resource Management**: CMDB asset management and service topology visualization

### Technology Stack

- **Backend**: Go + Gin + GORM + Redis + MySQL
- **Frontend**: Vue 3 + TypeScript + Ant Design Vue
- **AI Module**: Python + FastAPI + scikit-learn
- **Infrastructure**: Kubernetes + Prometheus + Grafana

### Project Components

This project consists of three main repositories:

- **[AI-CloudOps](https://github.com/GoSimplicity/AI-CloudOps)** - Core backend service
- **[AI-CloudOps-web](https://github.com/GoSimplicity/AI-CloudOps-web)** - Frontend interface
- **[AI-CloudOps-aiops](https://github.com/GoSimplicity/AI-CloudOps-aiops)** - AI intelligent analysis module

## Feature Modules

### Intelligent Operations (AIOps)

- Smart Monitoring: Machine learning-based monitoring data analysis
- Anomaly Detection: Automatic identification of system anomalies and performance issues
- Fault Prediction: Predictive maintenance and early warning
- Auto-Recovery: Rule-based and AI-powered intelligent recovery
- Root Cause Analysis: Rapid fault root cause identification

### Permission Management (RBAC)

- Multi-tenant Architecture: Support for multi-organizational isolation
- Role Management: Flexible role permission configuration
- Fine-grained Permissions: Operation-level permission control
- Audit Logging: Complete operational audit records

### Resource Management (CMDB)

- Service Tree: Visual resource topology structure
- Asset Discovery: Automated resource discovery and management
- Dependency Mapping: Service dependency graph management
- Tag Management: Flexible resource classification and tagging

### Ticketing System

- Ticket Templates: Customizable ticket types
- Workflow Engine: Automated approval workflows
- SLA Management: Service Level Agreement monitoring
- Collaboration Tools: Built-in communication and collaboration features

### Monitoring & Alerting

- Multi-dimensional Monitoring: Comprehensive system performance monitoring
- Real-time Alerting: Millisecond-level alert response
- Smart Thresholds: AI-powered dynamic threshold adjustment
- Alert Aggregation: Intelligent alert consolidation and noise reduction

### Kubernetes Management

- Cluster Management: Unified multi-cluster management
- Application Deployment: Visual application deployment and updates
- Resource Monitoring: Pod, Node, and Service monitoring
- Auto-scaling: HPA/VPA automatic scaling
- Configuration Management: ConfigMap and Secret management

## Live Demo

**Demo URL**: <http://68.64.177.180>

**Login Credentials**:

- Username: `demo`
- Password: `Demo@2025`

*Note: Demo environment is for testing only. Do not upload sensitive information*

## Core Contributors

Thanks to the following developers for their important contributions to the project:

- **[GoSimplicity](https://github.com/GoSimplicity)** - Project founder and core maintainer
- **[Penge666](https://github.com/Penge666)** - Senior developer
- **[shixiaocaia](https://github.com/shixiaocaia)** - Core contributor
- **[daihao4371](https://github.com/daihao4371)** - Feature developer

[![Contributors](https://contrib.rocks/image?repo=GoSimplicity/AI-CloudOps)](https://github.com/GoSimplicity/AI-CloudOps/graphs/contributors)

## Quick Start

### Requirements

- Go 1.21+
- Node.js 21.x
- pnpm latest
- Docker & Docker Compose
- Python 3.11+ (for AI module)

### Environment Check

```bash
go version          # >= 1.21
node --version      # >= 21.0
pnpm --version      # latest
docker --version    # latest
python3 --version   # >= 3.11
```

### Get the Code

```bash
# Clone backend project
git clone https://github.com/GoSimplicity/AI-CloudOps.git
cd AI-CloudOps

# Clone frontend project
git clone https://github.com/GoSimplicity/AI-CloudOps-web.git

# Clone AI module project
git clone https://github.com/GoSimplicity/AI-CloudOps-aiops.git
```

### Development Environment Setup

#### 1. Start Infrastructure Services

```bash
cd AI-CloudOps
# Start database and middleware
docker-compose -f docker-compose-env.yaml up -d

# Configure environment variables
cp env.example .env

# Check service status
docker-compose -f docker-compose-env.yaml ps
```

#### 2. Start Frontend Service

```bash
cd AI-CloudOps-web
# Install dependencies
pnpm install

# Start development server
pnpm run dev
```

Frontend will be available at [http://localhost:3000](http://localhost:3000)

#### 3. Start Backend Service

```bash
cd AI-CloudOps
# Install dependencies
go mod tidy

# Start backend service
go run main.go
```

Backend service endpoints:

- API Service: [http://localhost:8000](http://localhost:8000)

#### 4. Start AI Service (Optional)

```bash
cd AI-CloudOps-aiops
# Configure environment variables
cp env.example .env

# Install dependencies
pip install -r requirements.txt

# Train initial model
cd data/ && python machine-learning.py && cd ..

# Start AI service
python app/main.py
```

AI service will be available at [http://localhost:8001](http://localhost:8001)

## Production Deployment

### Docker Compose Deployment (Recommended)

```bash
cd AI-CloudOps
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f
```

### Kubernetes Deployment

```bash
cd deploy/kubernetes/
# Configure environment variables
cp config.example config

# Deploy to cluster
kubectl apply -f .

# Check deployment status
kubectl get pods,svc,ingress -l app=ai-cloudops
```

## Project Architecture

### Backend Structure

```text
AI-CloudOps/
├── cmd/                    # Command line tools
├── config/                 # Configuration files
├── internal/               # Core business logic
│   ├── middleware/         # Middleware
│   ├── model/              # Data models
│   ├── k8s/                # Kubernetes management
│   ├── user/               # User management
│   ├── prometheus/         # Monitoring module
│   ├── workorder/          # Ticketing system
│   ├── tree/               # Service tree CMDB
│   └── system/             # System management
├── pkg/                    # Common packages
├── docs/                   # API documentation
└── deploy/                 # Deployment configurations
```

### Frontend Structure

```text
AI-CloudOps-web/
├── apps/web-antd/          # Main application
│   ├── src/
│   │   ├── api/            # API interfaces
│   │   ├── components/     # Components
│   │   ├── views/          # Pages
│   │   ├── router/         # Routing
│   │   └── store/          # State management
│   └── dist/               # Build artifacts
└── packages/               # Shared packages
```

### AI Module Structure

```text
AI-CloudOps-aiops/
├── app/                    # Application code
│   ├── api/                # API routes
│   ├── core/               # Core logic
│   ├── models/             # Data models
│   └── services/           # Business services
├── data/                   # Data storage
├── config/                 # Configuration files
└── tests/                  # Test cases
```

## Contributing

### Contribution Process

1. Fork this project
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Create a Pull Request

### Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

```text
<type>(<scope>): <description>

[optional body]
[optional footer]
```

**Commit Types**:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code formatting
- `refactor`: Code refactoring
- `test`: Test related
- `ci`: CI/CD related

### Code Standards

- Follow existing code style
- Add necessary comments and documentation
- Ensure test coverage
- Follow best practices

## License

This project is licensed under the [MIT License](LICENSE).

## Contact Us

- **Email**: <bamboocloudops@gmail.com>
- **WeChat**: GoSimplicity (please note "AI-CloudOps" when adding)
- **GitHub**: [Submit Issue](https://github.com/GoSimplicity/AI-CloudOps/issues)

### WeChat Discussion Group

![WeChat Group QR Code](https://github.com/user-attachments/assets/6aa7bd2f-ee80-40d0-9301-0fd62292e82f)

## Acknowledgements

Thanks to all developers and users who contributed to this project, as well as the following open source projects:

- [Go](https://golang.org/) - High-performance backend language
- [Vue.js](https://vuejs.org/) - Progressive frontend framework
- [Kubernetes](https://kubernetes.io/) - Container orchestration platform
- [Prometheus](https://prometheus.io/) - Monitoring system
- [Ant Design Vue](https://antdv.com/) - Enterprise UI component library

---

**If you find this project helpful, please give us a Star ⭐**

[![Star History Chart](https://api.star-history.com/svg?repos=GoSimplicity/AI-CloudOps&type=Date)](https://star-history.com/#GoSimplicity/AI-CloudOps&Date)
