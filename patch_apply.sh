#!/bin/bash
awk '
/var existing string/ { skip=1 }
/case !isNoRows\(err\):/ {
    getline
    getline
    skip=0
    next
}
{ if (!skip) print }
' internal/database/migrate.go > temp.go && mv temp.go internal/database/migrate.go
