#!/bin/bash
awk '
/if err := rejectUnknownMigrations\(ctx, conn, known\); err != nil {/ {
    print
    getline
    print
    getline
    print
    print "\texistingMap := make(map[string]string)"
    print "\trows, err := conn.QueryContext(ctx, \"SELECT id, checksum FROM schema_migrations\")"
    print "\tif err == nil {"
    print "\t\tfor rows.Next() {"
    print "\t\t\tvar id, checksum string"
    print "\t\t\tif err := rows.Scan(&id, &checksum); err == nil {"
    print "\t\t\t\texistingMap[id] = checksum"
    print "\t\t\t}"
    print "\t\t}"
    print "\t\trows.Close()"
    print "\t}"
    print "\tvar pending []string"
    print "\tfor _, migrationID := range migrations {"
    print "\t\tcontent, err := fs.ReadFile(migrationFS, fmt.Sprintf(\"%s/%s.sql\", migrationID, dialect))"
    print "\t\tif err != nil {"
    print "\t\t\treturn fmt.Errorf(\"missing migration script %s/%s.sql: %w\", migrationID, dialect, err)"
    print "\t\t}"
    print "\t\tchecksum, err := migrationChecksum(migrationFS, migrationID, dialect, content)"
    print "\t\tif err != nil {"
    print "\t\t\treturn err"
    print "\t\t}"
    print "\t\tif existingChecksum, ok := existingMap[migrationID]; ok {"
    print "\t\t\tif existingChecksum != checksum {"
    print "\t\t\t\treturn fmt.Errorf(\"migration %s checksum mismatch\", migrationID)"
    print "\t\t\t}"
    print "\t\t\tcontinue"
    print "\t\t}"
    print "\t\tpending = append(pending, migrationID)"
    print "\t}"
    print "\tmigrations = pending"
    next
}
{ print }
' internal/database/migrate.go > temp.go && mv temp.go internal/database/migrate.go

awk '
/func applyMigration\(/ { inFunc = 1 }
inFunc && /var existing string/ {
    skip = 1
}
inFunc && skip && /case !isNoRows\(err\):/ {
    getline
    getline
    skip = 0
    inFunc = 0
    next
}
{ if (!skip) print }
' internal/database/migrate.go > temp.go && mv temp.go internal/database/migrate.go
