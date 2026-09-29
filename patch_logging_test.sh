#!/bin/bash
sed -i 's/`"session_id":"session-1"`, //g' internal/observability/logging_test.go
sed -i 's/`"actor_id":"actor-1"`, //g' internal/observability/logging_test.go
