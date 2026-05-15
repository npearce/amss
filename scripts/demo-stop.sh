#!/usr/bin/env bash
echo "Stopping AMSS demo..."

# Stop activity generator
pkill -f "activity-generator" 2>/dev/null && echo "  Activity generator stopped." || echo "  Activity generator was not running."

# Stop all port-forwards
pkill -f "port-forward" 2>/dev/null && echo "  Port-forwards stopped." || echo "  No port-forwards running."

echo ""
echo "Demo stopped. Lab environment is still running."
echo "To restart a demo:  ./scripts/demo-track{1-4}.sh"
echo "To teardown the lab: ./scripts/teardown.sh"
