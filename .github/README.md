# CI/CD Pipeline Documentation

This repository uses GitHub Actions for continuous integration and deployment.

## Workflows

### Development Pipeline (`develop.yml`)

**Triggers:**
- Push to `develop` branch
- Pull requests to `develop` branch

**Jobs:**
1. **Test** - Runs on all triggers
   - Go 1.23 setup with module caching
   - Run tests: `go test -v ./...`
   - Run linting: `golangci-lint` with 5m timeout

2. **Build and Push** - Only on push to develop
   - Build Docker image `credo7/reservia-admin-api`
   - Push with tags: `develop`, `develop-<sha>`, `dev-latest`
   - Multi-platform build (linux/amd64, linux/arm64)

3. **Deploy** - Only on push to develop
   - Deploy to `reservia-dev` namespace using Helm
   - Uses `dev.properties.yaml` values
   - 5-minute deployment timeout

### Production Pipeline (`production.yml`)

**Triggers:**
- Push to `production` branch
- GitHub releases

**Jobs:**
1. **Test** - Same as development

2. **Build and Push** - Enhanced for production
   - Same as development, plus semantic versioning
   - Additional tags: `v1.0.0`, `1.0`, `prod-latest`
   - **Security scanning** with Trivy vulnerability scanner
   - SARIF results uploaded to GitHub Security tab

3. **Deploy** - Production deployment
   - Deploy to `reservia-prod` namespace using Helm
   - Uses `prod.properties.yaml` values (with secrets)
   - 10-minute deployment timeout
   - Extended health checks with `kubectl wait`
   - **Telegram notifications** on success/failure

## Required Secrets

Configure these secrets in your GitHub repository settings:

### Docker Hub
- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password/token

### Kubernetes
- `KUBE_CONFIG_DEV` - Base64-encoded kubeconfig for development cluster
- `KUBE_CONFIG_PROD` - Base64-encoded kubeconfig for production cluster

### Telegram Notifications (Production Only)
- `TELEGRAM_BOT_TOKEN` - Bot token for deployment notifications
- `TELEGRAM_CHAT_ID` - Chat ID for notifications

## Environment Setup

### Development Environment
- **Namespace:** `reservia-dev`
- **App Name:** `admin-api`
- **Values:** `charts/dev.properties.yaml`
- **Replicas:** 1
- **Resources:** 100m-500m CPU, 128Mi-512Mi memory

### Production Environment
- **Namespace:** `reservia-prod`  
- **App Name:** `admin-api`
- **Values:** `charts/prod.properties.yaml`
- **Replicas:** 2+ (with auto-scaling)
- **Resources:** 200m-1000m CPU, 256Mi-1Gi memory

## Port Configuration

**Admin API uses port 8000** (same as client-api for consistency):
- Internal container port: 8000
- Kubernetes service port: 80 → 8000
- Health checks on port 8000

## Docker Image Strategy

**Registry:** `docker.io/credo7/reservia-admin-api`

**Development Tags:**
- `develop` - Latest develop branch
- `develop-<sha>` - Specific commit
- `dev-latest` - Latest development build

**Production Tags:**
- `production` - Latest production branch  
- `production-<sha>` - Specific commit
- `v1.0.0` - Semantic version (from releases)
- `1.0` - Major.minor version
- `prod-latest` - Latest production build

## Deployment Process

### Development
1. Push code to `develop` branch
2. Pipeline runs tests
3. If tests pass, builds and pushes Docker image
4. Deploys to `reservia-dev` namespace
5. Verifies deployment with pod logs

### Production
1. Push code to `production` branch OR create GitHub release
2. Pipeline runs tests
3. If tests pass, builds and pushes Docker image
4. **Security scan** with Trivy
5. Deploys to `reservia-prod` namespace with secrets
6. **Extended verification** with health checks
7. **Telegram notification** of deployment status

## Monitoring and Verification

Both pipelines include deployment verification:
- List pods in target namespace
- Show recent pod logs
- Production includes `kubectl wait` for ready state

## Security Features

- **Multi-stage Docker builds** with scratch base image
- **Non-root user** execution (UID 65534)
- **Vulnerability scanning** in production
- **Secrets management** for production deployments
- **SARIF security reports** uploaded to GitHub

## Troubleshooting

### Build Failures
- Check Go version compatibility (requires 1.23)
- Verify all dependencies are in go.mod
- Check golangci-lint issues

### Deployment Failures
- Verify kubeconfig secrets are correctly configured
- Check namespace exists and has proper permissions
- Ensure Kubernetes secrets exist for production
- Check Helm chart values for syntax errors

### Docker Issues
- Verify Docker Hub credentials
- Check if image tags are valid
- Ensure Dockerfile builds successfully locally