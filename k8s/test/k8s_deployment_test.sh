#!/bin/bash
set -e

# Kubernetes Deployment Test Script
# Tests Phase 6 Kubernetes deployment

echo "========================================="
echo "Phase 6: Kubernetes Deployment Tests"
echo "========================================="

NAMESPACE="transcode"
TIMEOUT=300

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Test 1: Namespace exists
test_namespace() {
    info "Testing namespace..."
    if kubectl get namespace $NAMESPACE > /dev/null 2>&1; then
        success "Namespace $NAMESPACE exists"
        return 0
    else
        error "Namespace $NAMESPACE does not exist"
        return 1
    fi
}

# Test 2: All deployments are ready
test_deployments() {
    info "Testing deployments..."

    DEPLOYMENTS=("api" "worker-cpu")

    for deploy in "${DEPLOYMENTS[@]}"; do
        if kubectl get deployment $deploy -n $NAMESPACE > /dev/null 2>&1; then
            READY=$(kubectl get deployment $deploy -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
            DESIRED=$(kubectl get deployment $deploy -n $NAMESPACE -o jsonpath='{.spec.replicas}')

            if [ "$READY" = "$DESIRED" ]; then
                success "Deployment $deploy is ready ($READY/$DESIRED)"
            else
                error "Deployment $deploy is not ready ($READY/$DESIRED)"
                return 1
            fi
        else
            error "Deployment $deploy does not exist"
            return 1
        fi
    done

    return 0
}

# Test 3: StatefulSets are ready
test_statefulsets() {
    info "Testing statefulsets..."

    STATEFULSETS=("postgres" "redis" "rabbitmq")

    for sts in "${STATEFULSETS[@]}"; do
        if kubectl get statefulset $sts -n $NAMESPACE > /dev/null 2>&1; then
            READY=$(kubectl get statefulset $sts -n $NAMESPACE -o jsonpath='{.status.readyReplicas}')
            DESIRED=$(kubectl get statefulset $sts -n $NAMESPACE -o jsonpath='{.spec.replicas}')

            if [ "$READY" = "$DESIRED" ]; then
                success "StatefulSet $sts is ready ($READY/$DESIRED)"
            else
                error "StatefulSet $sts is not ready ($READY/$DESIRED)"
                return 1
            fi
        else
            error "StatefulSet $sts does not exist"
            return 1
        fi
    done

    return 0
}

# Test 4: Services are accessible
test_services() {
    info "Testing services..."

    SERVICES=("api-service" "postgres-service" "redis-service" "rabbitmq-service")

    for svc in "${SERVICES[@]}"; do
        if kubectl get service $svc -n $NAMESPACE > /dev/null 2>&1; then
            success "Service $svc exists"
        else
            error "Service $svc does not exist"
            return 1
        fi
    done

    return 0
}

# Test 5: ConfigMaps exist
test_configmaps() {
    info "Testing configmaps..."

    CONFIGMAPS=("transcode-config" "postgres-config" "redis-config" "rabbitmq-config")

    for cm in "${CONFIGMAPS[@]}"; do
        if kubectl get configmap $cm -n $NAMESPACE > /dev/null 2>&1; then
            success "ConfigMap $cm exists"
        else
            error "ConfigMap $cm does not exist"
            return 1
        fi
    done

    return 0
}

# Test 6: Secrets exist
test_secrets() {
    info "Testing secrets..."

    if kubectl get secret transcode-secrets -n $NAMESPACE > /dev/null 2>&1; then
        success "Secret transcode-secrets exists"
        return 0
    else
        error "Secret transcode-secrets does not exist"
        return 1
    fi
}

# Test 7: HPA is configured
test_hpa() {
    info "Testing HorizontalPodAutoscaler..."

    HPAS=("api-hpa" "worker-cpu-hpa")

    for hpa in "${HPAS[@]}"; do
        if kubectl get hpa $hpa -n $NAMESPACE > /dev/null 2>&1; then
            success "HPA $hpa exists"
        else
            error "HPA $hpa does not exist"
            return 1
        fi
    done

    return 0
}

# Test 8: Network policies exist
test_network_policies() {
    info "Testing network policies..."

    POLICIES=("default-deny-all" "api-network-policy" "worker-network-policy")

    for policy in "${POLICIES[@]}"; do
        if kubectl get networkpolicy $policy -n $NAMESPACE > /dev/null 2>&1; then
            success "NetworkPolicy $policy exists"
        else
            error "NetworkPolicy $policy does not exist"
            return 1
        fi
    done

    return 0
}

# Test 9: Monitoring stack
test_monitoring() {
    info "Testing monitoring stack..."

    MONITORING_PODS=("prometheus" "grafana" "alertmanager")

    for pod in "${MONITORING_PODS[@]}"; do
        if kubectl get deployment $pod -n $NAMESPACE > /dev/null 2>&1; then
            success "Monitoring component $pod exists"
        else
            error "Monitoring component $pod does not exist"
            return 1
        fi
    done

    return 0
}

# Test 10: API health check
test_api_health() {
    info "Testing API health..."

    # Get API service endpoint
    API_SERVICE=$(kubectl get svc api-service -n $NAMESPACE -o jsonpath='{.spec.clusterIP}')

    if [ -n "$API_SERVICE" ]; then
        # Port forward to test
        kubectl port-forward -n $NAMESPACE svc/api-service 8080:80 > /dev/null 2>&1 &
        PF_PID=$!
        sleep 3

        if curl -f http://localhost:8080/health > /dev/null 2>&1; then
            success "API health check passed"
            kill $PF_PID
            return 0
        else
            error "API health check failed"
            kill $PF_PID
            return 1
        fi
    else
        error "API service IP not found"
        return 1
    fi
}

# Test 11: Prometheus metrics
test_prometheus_metrics() {
    info "Testing Prometheus metrics..."

    # Port forward to Prometheus
    kubectl port-forward -n $NAMESPACE svc/prometheus 9090:9090 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 3

    if curl -f http://localhost:9090/api/v1/targets > /dev/null 2>&1; then
        success "Prometheus is accessible"
        kill $PF_PID
        return 0
    else
        error "Prometheus is not accessible"
        kill $PF_PID
        return 1
    fi
}

# Test 12: PersistentVolumeClaims are bound
test_pvcs() {
    info "Testing PersistentVolumeClaims..."

    PVCS=$(kubectl get pvc -n $NAMESPACE -o jsonpath='{.items[*].metadata.name}')

    for pvc in $PVCS; do
        STATUS=$(kubectl get pvc $pvc -n $NAMESPACE -o jsonpath='{.status.phase}')
        if [ "$STATUS" = "Bound" ]; then
            success "PVC $pvc is bound"
        else
            error "PVC $pvc is not bound (status: $STATUS)"
            return 1
        fi
    done

    return 0
}

# Run all tests
run_all_tests() {
    TOTAL=0
    PASSED=0

    tests=(
        "test_namespace"
        "test_configmaps"
        "test_secrets"
        "test_pvcs"
        "test_statefulsets"
        "test_services"
        "test_deployments"
        "test_hpa"
        "test_network_policies"
        "test_monitoring"
        "test_api_health"
        "test_prometheus_metrics"
    )

    for test in "${tests[@]}"; do
        echo ""
        TOTAL=$((TOTAL + 1))
        if $test; then
            PASSED=$((PASSED + 1))
        fi
    done

    echo ""
    echo "========================================="
    echo "Test Results: $PASSED/$TOTAL passed"
    echo "========================================="

    if [ $PASSED -eq $TOTAL ]; then
        success "All tests passed!"
        return 0
    else
        error "Some tests failed"
        return 1
    fi
}

# Main
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [test_name]"
    echo ""
    echo "Available tests:"
    echo "  test_namespace"
    echo "  test_deployments"
    echo "  test_statefulsets"
    echo "  test_services"
    echo "  test_configmaps"
    echo "  test_secrets"
    echo "  test_hpa"
    echo "  test_network_policies"
    echo "  test_monitoring"
    echo "  test_api_health"
    echo "  test_prometheus_metrics"
    echo "  test_pvcs"
    echo ""
    echo "Run without arguments to execute all tests"
    exit 0
fi

if [ -n "$1" ]; then
    # Run specific test
    $1
else
    # Run all tests
    run_all_tests
fi
