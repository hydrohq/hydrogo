import re

filepath = 'modules/members/transport/http.go'
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

content = re.sub(r'\bwriteMemberMobileJSON\(w, ', r'api.WriteJSON(w, ', content)
content = re.sub(r'\bwriteMemberMobileError\(w, ([^,]+), ([^,]+), ([^\)]+)\)', r'api.WriteError(w, r, \1, \3, nil)', content)

if content != orig_content:
    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)
    print("Updated")
