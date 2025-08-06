# Reservia Admin API Helm Chart

This Helm chart deploys the Reservia Admin API service to a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- MongoDB database
- RabbitMQ (optional, for email notifications)

## Installation

### Development Environment

```bash
helm install reservia-admin-api . -f dev.properties.yaml
```

### Production Environment

First, create the necessary secrets:

```bash
# MongoDB credentials
kubectl create secret generic mongodb-credentials \
  --from-literal=uri="mongodb://username:password@mongo-host:27017/reservia_prod"

# Auth secret
kubectl create secret generic auth-secret \
  --from-literal=jwt_secret="your-production-jwt-secret"

# RabbitMQ credentials (optional)
kubectl create secret generic rabbitmq-credentials \
  --from-literal=uri="amqp://username:password@rabbitmq-host:5672/"
```

Then deploy:

```bash
helm install reservia-admin-api . -f prod.properties.yaml
```

## Configuration

### Required Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `MONGODB_URI` | MongoDB connection string | Required |
| `DATABASE_NAME` | MongoDB database name | Required |
| `JWT_SECRET` | JWT signing secret | Required |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_LEVEL` | Logging level | `info` |
| `RABBITMQ_URI` | RabbitMQ connection string | Optional |
| `RABBITMQ_QUEUE_NAME` | Email notification queue | `email_notifications` |

## Monitoring

The chart includes support for Prometheus monitoring via ServiceMonitor (when enabled).

## Health Checks

The service exposes a health endpoint at `/health` which is used for both liveness and readiness probes.

## Scaling

Production deployments support horizontal pod autoscaling based on CPU and memory usage.

## Security

- Runs as non-root user (UID 65534)
- Uses secrets for sensitive configuration in production
- Supports pod anti-affinity for better availability