#!/usr/bin/env bash
set -euo pipefail

echo "🛠️  Building proto stubs..."
make proto

echo "🚀 Bringing up infra..."
make up

echo "✅ Infra is up. Use 'docker-compose ps' to verify."