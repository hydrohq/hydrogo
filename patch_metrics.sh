#!/bin/bash
sed -i 's/path := normalizeRoute(r.URL.Path)/path := r.URL.Path\n\tif r.Pattern != "" {\n\t\tpath = r.Pattern\n\t} else {\n\t\tpath = normalizeRoute(r.URL.Path)\n\t}/g' internal/observability/metrics.go
