#!/bin/bash
# Returns version string based on git state

if git describe --tags 2>/dev/null >/dev/null; then
  git describe --tags --always --dirty
else
  COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  if git diff --quiet 2>/dev/null; then
    echo "v0.0.0-g${COMMIT}"
  else
    echo "v0.0.0-g${COMMIT}-dirty"
  fi
fi