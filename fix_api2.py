import os
import re

def process_file(filepath):
    if not os.path.exists(filepath):
        print(f"File not found: {filepath}")
        return

    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    orig_content = content

    if 'github.com/ardaozyildirim/hydrogo/internal/api' not in content:
        content = re.sub(
            r'(import \(\n(?:\s+".*?"\n)+)',
            r'\1\t"github.com/ardaozyildirim/hydrogo/internal/api"\n',
            content,
            count=1
        )

    content = re.sub(r'\bwriteJSON\(w, ', r'api.WriteJSON(w, ', content)
    content = re.sub(r'func writeJSON\(w http\.ResponseWriter, status int, value any\) \{[\s\S]*?\}\n\n?', '', content)

    if content != orig_content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"Updated {filepath}")
    else:
        print(f"No changes for {filepath}")

process_file('internal/auth/http.go')
process_file('internal/collections/transport/http.go')
process_file('internal/media/http.go')
