import os
import re

def process_file(filepath, err_status_mapper=None):
    if not os.path.exists(filepath):
        print(f"File not found: {filepath}")
        return

    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    orig_content = content

    # Add api import if needed
    if 'github.com/ardaozyildirim/hydrogo/internal/api' not in content:
        content = re.sub(
            r'(import \(\n(?:\s+".*?"\n)+)',
            r'\1\t"github.com/ardaozyildirim/hydrogo/internal/api"\n',
            content,
            count=1
        )

    # Replace writeJSON
    content = re.sub(r'\bwriteJSON\(w, ', r'api.WriteJSON(w, ', content)
    # Remove writeJSON func
    content = re.sub(r'func writeJSON\(w http\.ResponseWriter, status int, value any\) \{[\s\S]*?\}\n\n?', '', content)

    # Specifics for members/mobile.go
    if filepath.endswith('mobile.go'):
        content = re.sub(r'\bwriteMemberMobileJSON\(w, ', r'api.WriteJSON(w, ', content)
        content = re.sub(r'func writeMemberMobileJSON\(w http\.ResponseWriter, status int, value any\) \{[\s\S]*?\}\n\n?', '', content)
        # Note: writeMemberMobileError takes (w, status, code, message)
        content = re.sub(r'\bwriteMemberMobileError\(w, ([^,]+), ([^,]+), ([^\)]+)\)', r'api.WriteError(w, r, \1, \3, nil)', content)
        content = re.sub(r'func writeMemberMobileError\(w http\.ResponseWriter, status int, code, message string\) \{[\s\S]*?\}\n\n?', '', content)

    if err_status_mapper:
        content = err_status_mapper(content)

    if content != orig_content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"Updated {filepath}")
    else:
        print(f"No changes for {filepath}")

def map_news_err(content):
    # writeError in news
    # old signature: func writeError(w http.ResponseWriter, err error)
    # we replace call `writeError(w, err)` with:
    # api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
    content = re.sub(r'\bwriteError\(w, err\)', r'api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)', content)
    
    # remove old writeError
    content = re.sub(r'func writeError\(w http\.ResponseWriter, err error\) \{[\s\S]*?http\.Error\(w, "news operation failed", status\)\n\}\n\n?', '', content)
    
    # add newsErrorStatus
    helper = """
func newsErrorStatus(err error) int {
	if errors.Is(err, news.ErrRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, news.ErrInvalidItem) {
		return http.StatusBadRequest
	}
	if errors.Is(err, news.ErrConflict) || errors.Is(err, news.ErrScheduleStale) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}
"""
    if "func newsErrorStatus" not in content:
        content += helper

    return content

process_file('modules/news/transport/http.go', map_news_err)
process_file('modules/pages/transport/http.go')
process_file('modules/members/transport/http.go')
process_file('modules/members/transport/mobile.go')
