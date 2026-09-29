#!/bin/bash
awk '
/tx, err := db.BeginTx\(ctx, nil\)/ {
    print
    getline
    print
    getline
    print
    getline
    print
    print "\tdefer tx.Rollback()"
    next
}
/reader, err := db.BeginTx\(ctx, nil\)/ {
    print
    getline
    print
    getline
    print
    getline
    print
    print "\tdefer reader.Rollback()"
    next
}
{ print }
' internal/database/maintenance_test.go > temp.go && mv temp.go internal/database/maintenance_test.go
