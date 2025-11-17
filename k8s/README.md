# Kubernetes Deployment

This directory contains Kubernetes manifests and Helm charts for deploying the transcode service.

## Directory Structure

```
k8s/
├── base/                  # Base Kubernetes manifests
│   ├── namespace.yaml     # Namespace definition
│   ├── configmap.yaml     # Configuration
│   ├── secret.yaml        # Secrets (update before deploying!)
│   ├── postgres-statefulset.yaml
│   ├── redis-statefulset.yaml
│   ├── rabbitmq-statefulset.yaml
│   ├── api-deployment.yaml
│   ├── worker-deployment.yaml
│   ├── ingress.yaml       # Ingress configuration
│   ├── hpa.yaml           # HorizontalPodAutoscaler
│   ├── network-policy.yaml # Network security policies
│   ├── backup-cronjob.yaml # Automated backups
│   ├── disaster-recovery.md # DR procedures
│   └── kustomization.yaml  # Kustomize config
├── monitoring/            # Monitoring stack
│   ├── prometheus-deployment.yaml
│   ├── grafana-deployment.yaml
│   ├── grafana-dashboards.yaml
│   ├── alertmanager-deployment.yaml
│   └── jaeger-deployment.yaml
├── helm/                  # Helm charts
│   └── transcode/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
└── test/                  # Test scripts
    └── k8s_deployment_test.sh
```

## Quick Start

### Option 1: Using Helm (Recommended)

```bash
# Install
helm install transcode ./helm/transcode \
  --namespace transcode \
  --create-namespace \
  --set secrets.postgres.password=YOUR_PASSWORD

# Upgrade
helm upgrade transcode ./helm/transcode \
  --namespace transcode

# Uninstall
helm uninstall transcode -n transcode
```

### Option 2: Using kubectl + Kustomize

```bash
# Deploy base resources
kubectl apply -k base/

# Deploy monitoring
kubectl apply -f monitoring/

# Verify
kubectl get all -n transcode
```

### Option 3: Using kubectl directly

```bash
# Create namespace
kubectl apply -f base/namespace.yaml

# Update secrets (IMPORTANT!)
# Edit base/secret.yaml first!
kubectl apply -f base/secret.yaml

# Deploy in order
kubectl apply -f base/configmap.yaml
kubectl apply -f base/postgres-statefulset.yaml
kubectl apply -f base/redis-statefulset.yaml
kubectl apply -f base/rabbitmq-statefulset.yaml
kubectl apply -f base/api-deployment.yaml
kubectl apply -f base/worker-deployment.yaml
kubectl apply -f base/ingress.yaml
kubectl apply -f base/hpa.yaml
kubectl apply -f base/network-policy.yaml
kubectl apply -f base/backup-cronjob.yaml
kubectl apply -f monitoring/
```

## Configuration

### Update Secrets

**IMPORTANT**: Before deploying, update `base/secret.yaml` with your actual credentials:

```bash
# Edit the file
vi base/secret.yaml

# Update these values:
# - POSTGRES_PASSWORD
# - AWS_ACCESS_KEY_ID
# - AWS_SECRET_ACCESS_KEY
# - JWT_SECRET
# - RABBITMQ_PASSWORD
# - RABBITMQ_ERLANG_COOKIE
# - WEBHOOK_SECRET
```

### Update ConfigMap

Edit `base/configmap.yaml` to configure:
- S3 bucket name
- Database settings
- Transcoding settings
- Monitoring endpoints

### Update Ingress

Edit `base/ingress.yaml` to configure:
- Domain names
- TLS certificates
- Rate limiting

## Verification

```bash
# Check pods
kubectl get pods -n transcode

# Check services
kubectl get svc -n transcode

# Check ingress
kubectl get ingress -n transcode

# Test API
curl https://api.your-domain.com/health

# View logs
kubectl logs -f deployment/api -n transcode
```

## Monitoring

### Access Grafana

```bash
# Get password
kubectl get secret -n transcode grafana-admin -o jsonpath="{.data.password}" | base64 --decode

# Port forward (or use ingress)
kubectl port-forward -n transcode svc/grafana 3000:3000

# Access at http://localhost:3000
# Username: admin
```

### Access Prometheus

```bash
kubectl port-forward -n transcode svc/prometheus 9090:9090
# Access at http://localhost:9090
```

### Access Jaeger

```bash
kubectl port-forward -n transcode svc/jaeger-query 16686:16686
# Access at http://localhost:16686
```

## Scaling

### Manual Scaling

```bash
# Scale API
kubectl scale deployment api -n transcode --replicas=5

# Scale workers
kubectl scale deployment worker-cpu -n transcode --replicas=10
```

### Auto-scaling

Auto-scaling is configured via HPA based on:
- CPU utilization
- Memory utilization
- Queue depth (custom metrics)

View HPA status:
```bash
kubectl get hpa -n transcode
```

## Updates

### Rolling Update

```bash
# Update image
kubectl set image deployment/api -n transcode \
  api=transcode/api:v6.1.0

# Check rollout
kubectl rollout status deployment/api -n transcode
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/api -n transcode
```

## Backups

### Manual Backup

```bash
# Trigger backup job
kubectl create job --from=cronjob/postgres-backup \
  postgres-backup-manual -n transcode

# Check status
kubectl get jobs -n transcode
```

### Restore

See `base/disaster-recovery.md` for complete restore procedures.

## Testing

```bash
# Run deployment tests
./test/k8s_deployment_test.sh

# Run specific test
./test/k8s_deployment_test.sh test_deployments
```

## Troubleshooting

### Pod Not Starting

```bash
# Describe pod
kubectl describe pod <pod-name> -n transcode

# Check logs
kubectl logs <pod-name> -n transcode

# Check events
kubectl get events -n transcode --sort-by='.lastTimestamp'
```

### Service Not Accessible

```bash
# Check service
kubectl get svc -n transcode

# Check endpoints
kubectl get endpoints -n transcode

# Debug from inside cluster
kubectl run -it debug --image=alpine --restart=Never -n transcode -- sh
```

### View Resource Usage

```bash
# Pod resource usage
kubectl top pods -n transcode

# Node resource usage
kubectl top nodes
```

## Production Checklist

Before deploying to production:

- [ ] Update all secrets in `base/secret.yaml`
- [ ] Configure ingress with your domain
- [ ] Set up TLS certificates (cert-manager)
- [ ] Configure S3 bucket for backups
- [ ] Set up AlertManager notifications
- [ ] Review resource requests/limits
- [ ] Test disaster recovery procedures
- [ ] Configure network policies
- [ ] Set up monitoring dashboards
- [ ] Test auto-scaling
- [ ] Configure backup retention
- [ ] Set up log aggregation
- [ ] Review security policies

## Documentation

- [Phase 6 Documentation](../PHASE6.md) - Complete Phase 6 guide
- [Disaster Recovery](base/disaster-recovery.md) - DR procedures
- [Helm Chart](helm/transcode/README.md) - Helm chart documentation

## Support

For issues or questions, see the main [README](../README.md) or open an issue on GitHub.
