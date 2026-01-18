#!/bin/bash
echo "Running TypeScript type check..."
bunx tsc --noEmit

if [ $? -eq 0 ]; then
  echo "✅ Type check passed"
else
  echo "❌ Type check failed"
  exit 1
fi