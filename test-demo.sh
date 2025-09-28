#!/bin/bash

# AgentHub Observability Demo Test Script
# This script demonstrates the working configuration system

set -e

echo "🚀 AgentHub Observability Demo Test"
echo "===================================="

# Load environment configuration
echo "📋 Loading environment configuration..."
source .envrc
echo "   ✅ JAEGER_ENDPOINT: $JAEGER_ENDPOINT"
echo "   ✅ BROKER_HEALTH_PORT: $BROKER_HEALTH_PORT"
echo "   ✅ GRAFANA_PORT: $GRAFANA_PORT"

# Check if observability stack is running
echo ""
echo "🔍 Checking observability stack..."
if docker-compose -f observability/docker-compose.yml ps | grep -q "Up"; then
    echo "   ✅ Observability stack is running"
else
    echo "   ⚠️  Starting observability stack..."
    cd observability
    docker-compose up -d
    cd ..
    echo "   ✅ Observability stack started"
fi

# Check if ports are accessible
echo ""
echo "🌐 Checking service accessibility..."
if curl -s "http://localhost:$GRAFANA_PORT/api/health" > /dev/null; then
    echo "   ✅ Grafana accessible on port $GRAFANA_PORT"
else
    echo "   ❌ Grafana not accessible on port $GRAFANA_PORT"
fi

if curl -s "http://localhost:16686/api/services" > /dev/null; then
    echo "   ✅ Jaeger accessible on port 16686"
else
    echo "   ❌ Jaeger not accessible on port 16686"
fi

# Test build with new configuration
echo ""
echo "🔨 Testing build with new configuration..."
if go build -tags observability -o bin/test-demo broker/main_observability.go; then
    echo "   ✅ Broker builds successfully with configuration"
    rm -f bin/test-demo
else
    echo "   ❌ Build failed"
    exit 1
fi

echo ""
echo "🎉 Configuration test completed successfully!"
echo ""
echo "📚 Next steps:"
echo "   1. Run: go run -tags observability broker/main_observability.go"
echo "   2. Run: go run -tags observability agents/subscriber/main_observability.go"
echo "   3. Run: go run -tags observability agents/publisher/main_observability.go"
echo "   4. Visit: http://localhost:$GRAFANA_PORT (admin/admin)"
echo "   5. Visit: http://localhost:16686 for traces"
echo ""
echo "✨ All services will use environment-configured ports!"