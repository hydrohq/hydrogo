import os
import re

def fix_imports():
    for root, dirs, files in os.walk('.'):
        for file in files:
            if file.endswith('.go'):
                filepath = os.path.join(root, file)
                with open(filepath, 'r', encoding='utf-8') as f:
                    content = f.read()
                
                if 'api.Write' in content and '"github.com/ardaozyildirim/hydrogo/internal/api"' not in content:
                    match = re.search(r'import \((.*?)\)', content, re.DOTALL)
                    if match:
                        new_import = match.group(0).replace('import (', 'import (\n\t"github.com/ardaozyildirim/hydrogo/internal/api"')
                        content = content.replace(match.group(0), new_import)
                    else:
                        match = re.search(r'import ".*?"', content)
                        if match:
                            new_import = match.group(0) + '\nimport "github.com/ardaozyildirim/hydrogo/internal/api"'
                            content = content.replace(match.group(0), new_import)
                    with open(filepath, 'w', encoding='utf-8') as f:
                        f.write(content)
                    print(f"Fixed imports in {filepath}")

fix_imports()
