# Disaster Recovery Procedures

## Overview

This document outlines the disaster recovery procedures for the Transcode service, including backup strategies, recovery procedures, and business continuity planning.

## Backup Strategy

### Automated Backups

1. **PostgreSQL Database**
   - **Frequency**: Daily at 2 AM UTC
   - **Retention**: 30 days
   - **Location**: S3 bucket `transcode-backups/postgres/`
   - **CronJob**: `postgres-backup`

2. **Redis Cache**
   - **Frequency**: Daily at 3 AM UTC
   - **Retention**: 7 days
   - **Location**: S3 bucket `transcode-backups/redis/`
   - **CronJob**: `redis-backup`

3. **Video Files**
   - **Strategy**: S3 versioning enabled
   - **Retention**: 90 days for deleted objects
   - **Cross-region replication**: Enabled to `us-west-2`

4. **Configuration**
   - **Strategy**: GitOps - all configs in Git
   - **Repository**: Committed to version control
   - **Secrets**: Managed via Kubernetes Secrets (encrypted at rest)

### Manual Backups

Before major changes:
```bash
# Database backup
kubectl exec -n transcode postgres-0 -- pg_dump -U postgres transcode > backup.sql

# Redis backup
kubectl exec -n transcode redis-0 -- redis-cli BGSAVE
```

## Recovery Procedures

### PostgreSQL Database Recovery

#### Full Database Restore

```bash
# 1. List available backups
aws s3 ls s3://transcode-backups/postgres/

# 2. Download backup
aws s3 cp s3://transcode-backups/postgres/transcode-db-YYYYMMDD-HHMMSS.sql.gz /tmp/

# 3. Scale down services that use the database
kubectl scale deployment -n transcode api --replicas=0
kubectl scale deployment -n transcode worker-cpu --replicas=0
kubectl scale deployment -n transcode worker-gpu --replicas=0

# 4. Drop existing database (CAUTION!)
kubectl exec -n transcode postgres-0 -- psql -U postgres -c "DROP DATABASE transcode;"
kubectl exec -n transcode postgres-0 -- psql -U postgres -c "CREATE DATABASE transcode;"

# 5. Restore from backup
gunzip -c /tmp/transcode-db-YYYYMMDD-HHMMSS.sql.gz | \
  kubectl exec -i -n transcode postgres-0 -- psql -U postgres transcode

# 6. Verify data
kubectl exec -n transcode postgres-0 -- psql -U postgres transcode -c "\dt"

# 7. Scale services back up
kubectl scale deployment -n transcode api --replicas=3
kubectl scale deployment -n transcode worker-cpu --replicas=2
kubectl scale deployment -n transcode worker-gpu --replicas=1
```

#### Point-in-Time Recovery (PITR)

For PITR, ensure WAL archiving is enabled:

```yaml
# PostgreSQL ConfigMap addition
postgresql.conf: |
  wal_level = replica
  archive_mode = on
  archive_command = 'aws s3 cp %p s3://transcode-backups/postgres-wal/%f'
```

Restore procedure:
```bash
# 1. Restore base backup
# 2. Apply WAL files up to the desired point
# 3. Use recovery.conf to specify recovery target
```

### Redis Recovery

```bash
# 1. Download backup
aws s3 cp s3://transcode-backups/redis/transcode-redis-YYYYMMDD-HHMMSS.rdb /tmp/dump.rdb

# 2. Scale down Redis
kubectl scale statefulset -n transcode redis --replicas=0

# 3. Copy backup to PV (requires pod with PV mounted)
kubectl run -n transcode redis-restore --image=redis:7-alpine --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"redis-restore","image":"redis:7-alpine","command":["sleep","3600"],"volumeMounts":[{"name":"redis-data","mountPath":"/data"}]}],"volumes":[{"name":"redis-data","persistentVolumeClaim":{"claimName":"redis-data-redis-0"}}]}}'

kubectl cp /tmp/dump.rdb transcode/redis-restore:/data/dump.rdb

kubectl delete pod -n transcode redis-restore

# 4. Scale Redis back up
kubectl scale statefulset -n transcode redis --replicas=1
```

### Complete Cluster Disaster Recovery

#### Prerequisites
- Access to the Git repository with all manifests
- Access to S3 backups
- New Kubernetes cluster provisioned
- DNS updated to point to new cluster

#### Recovery Steps

1. **Provision New Cluster**
```bash
# Using your IaC tool (Terraform, eksctl, etc.)
# Ensure GPU node pools if needed
```

2. **Install Prerequisites**
```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml

# Install Prometheus Operator (if using)
kubectl apply -f https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.68.0/bundle.yaml
```

3. **Deploy Base Infrastructure**
```bash
# Apply namespace
kubectl apply -f k8s/base/namespace.yaml

# Apply secrets (update values first!)
kubectl apply -f k8s/base/secret.yaml

# Apply configmaps
kubectl apply -f k8s/base/configmap.yaml
```

4. **Deploy Stateful Services**
```bash
# PostgreSQL
kubectl apply -f k8s/base/postgres-statefulset.yaml

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n transcode --timeout=300s

# Restore PostgreSQL backup (see above)

# Redis
kubectl apply -f k8s/base/redis-statefulset.yaml

# RabbitMQ
kubectl apply -f k8s/base/rabbitmq-statefulset.yaml
```

5. **Deploy Application Services**
```bash
# API
kubectl apply -f k8s/base/api-deployment.yaml

# Workers
kubectl apply -f k8s/base/worker-deployment.yaml

# Services
kubectl apply -f k8s/base/ingress.yaml
```

6. **Deploy Monitoring Stack**
```bash
kubectl apply -f k8s/monitoring/prometheus-deployment.yaml
kubectl apply -f k8s/monitoring/grafana-deployment.yaml
kubectl apply -f k8s/monitoring/alertmanager-deployment.yaml
kubectl apply -f k8s/monitoring/jaeger-deployment.yaml
```

7. **Verify Services**
```bash
# Check all pods are running
kubectl get pods -n transcode

# Check services
kubectl get svc -n transcode

# Test API endpoint
curl https://api.transcode.example.com/health
```

8. **Restore Data**
- Database restored in step 4
- Video files already in S3
- Verify job history and video metadata

## RTO and RPO

### Recovery Time Objective (RTO)

- **Database**: 30 minutes
- **Complete Cluster**: 2 hours
- **Single Service**: 5 minutes

### Recovery Point Objective (RPO)

- **Database**: 24 hours (daily backup)
- **Video Files**: 0 (real-time replication)
- **Configuration**: 0 (Git)

## Failure Scenarios

### Scenario 1: Single Pod Failure

**Detection**: Kubernetes health checks, Prometheus alerts

**Recovery**: Automatic (Kubernetes recreates pod)

**RTO**: 1-2 minutes

### Scenario 2: Node Failure

**Detection**: Kubernetes node status, Prometheus alerts

**Recovery**: Automatic (pods rescheduled to healthy nodes)

**RTO**: 2-5 minutes

### Scenario 3: Availability Zone Failure

**Detection**: Multiple pod failures, increased latency

**Recovery**:
1. Kubernetes reschedules pods to other AZs
2. Monitor pod distribution
3. Scale up if needed

**RTO**: 5-10 minutes

### Scenario 4: Database Corruption

**Detection**: Application errors, query failures

**Recovery**:
1. Identify last known good backup
2. Follow PostgreSQL recovery procedure
3. Validate data integrity

**RTO**: 1-2 hours
**RPO**: 24 hours

### Scenario 5: Complete Region Failure

**Detection**: All services unreachable

**Recovery**:
1. Activate DR region
2. Update DNS
3. Follow complete cluster recovery
4. Restore latest backups

**RTO**: 4-6 hours
**RPO**: 24 hours

## Testing

### Backup Validation

Monthly backup restore tests:
```bash
# Create test namespace
kubectl create namespace transcode-test

# Restore backup to test namespace
# Validate data integrity
# Delete test namespace
```

### DR Drill

Quarterly full DR drill:
1. Simulate region failure
2. Execute complete cluster recovery
3. Verify all services operational
4. Document lessons learned
5. Update procedures

## Monitoring and Alerts

### Backup Alerts

- Backup job failures
- Backup size anomalies
- Old backups not deleted

### Recovery Metrics

- Time to detect failure
- Time to initiate recovery
- Time to full recovery
- Data loss (if any)

## Contacts

### Emergency Contacts

- **On-call Engineer**: [PagerDuty]
- **Database Admin**: [Contact]
- **Cloud Ops**: [Contact]
- **Security**: [Contact]

### Escalation Path

1. On-call engineer (0-15 min)
2. Team lead (15-30 min)
3. Engineering manager (30-60 min)
4. CTO (60+ min)

## Runbooks

### Quick Reference

| Scenario | Runbook | RTO |
|----------|---------|-----|
| Pod crash | Auto-recovery | 2 min |
| Node failure | Auto-recovery | 5 min |
| DB restore | Section: PostgreSQL Recovery | 1 hour |
| Full DR | Section: Complete Cluster DR | 4 hours |

## Compliance

- Backups encrypted at rest (AES-256)
- Backups encrypted in transit (TLS)
- Access logs maintained for 1 year
- Backup retention meets compliance requirements

## Review and Updates

- Review quarterly
- Update after each incident
- Update after infrastructure changes
- Version control all changes
