#!/bin/sh
set -e

echo "🔄 Running database migrations..."
migrate -path /migrations -database "${DATABASE_URL}" up

echo "✅ Migrations completed successfully"
echo "🚀 Starting application..."

# entrypoint의 CMD 인자를 실행 ($@)
exec "$@"
