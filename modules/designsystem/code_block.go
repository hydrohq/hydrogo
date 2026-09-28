package designsystem

import (
	"strings"
	"unicode/utf8"
)

type CodeBlockProps struct {
	Code     string
	Language string
}

type CodeToken struct {
	Text string
	Kind string
}

// CodeTokens applies a small, dependency-free lexical highlighter. Unknown
// languages remain readable plain text; every token is escaped by Templ.
func CodeTokens(code, language string) []CodeToken {
	keywords := map[string]map[string]bool{
		"go":     words("break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var"),
		"js":     words("async await break case catch class const continue debugger default delete do else export extends finally for function if import in instanceof let new return super switch this throw try typeof var void while with yield"),
		"ts":     words("async await break case catch class const continue debugger default delete do else export extends finally for function if import in instanceof let new return super switch this throw try typeof var void while with yield interface type enum implements private public readonly"),
		"python": words("and as assert async await break class continue def del elif else except False finally for from global if import in is lambda None nonlocal not or pass raise return True try while with yield"),
		"py":     words("and as assert async await break class continue def del elif else except False finally for from global if import in is lambda None nonlocal not or pass raise return True try while with yield"),
		"sql":    words("all alter and as asc between by case cast column create cross current_date current_timestamp database desc distinct drop else end exists from full group having in index inner insert into is join left like limit not null offset on or order outer primary references right select set table then union unique update values when where"),
		"html":   words("DOCTYPE html head body title meta link script style div span section article main header footer"),
		"css":    words("important inherit initial unset none auto block flex grid relative absolute fixed sticky"),
		"rust":   words("as async await break const continue crate dyn else enum extern false fn for if impl in let loop match mod move mut pub ref return self Self static struct super trait true type unsafe use where while"),
		"java":   words("abstract assert boolean break byte case catch char class const continue default do double else enum extends final finally float for if implements import instanceof int interface long native new package private protected public return short static strictfp super switch synchronized this throw throws transient try void volatile while"),
		"c":      words("auto break case char const continue default do double else enum extern float for goto if inline int long register restrict return short signed sizeof static struct switch typedef union unsigned void volatile while"),
		"cpp":    words("alignas alignof auto bool break case catch char class const constexpr continue default delete do double else enum explicit extern false float for friend goto if inline int long namespace new nullptr operator private protected public return short signed sizeof static struct switch template this throw true try typedef typename union unsigned using virtual void volatile while"),
		"ruby":   words("alias and begin break case class def defined do else elsif end ensure false for if in module next nil not or redo rescue retry return self super then true undef unless until when while yield"),
		"php":    words("abstract and array as break callable case catch class clone const continue declare default do else elseif empty enddeclare endfor endforeach endif endswitch endwhile extends final finally fn for foreach function global if implements include include_once instanceof interface namespace new or private protected public require require_once return static switch throw trait try use var while xor"),
		"json":   words("true false null"),
	}
	language = NormalizeCodeLanguage(language)
	known := keywords[language]
	var tokens []CodeToken
	for i := 0; i < len(code); {
		start := i
		kind := ""
		if (code[i] == '/' && i+1 < len(code) && code[i+1] == '/') || (code[i] == '#' && (language == "python" || language == "py" || language == "shell" || language == "bash" || language == "sh" || language == "yaml" || language == "yml")) {
			for i < len(code) && code[i] != '\n' {
				i++
			}
			kind = "comment"
		} else if code[i] == '"' || code[i] == '\'' || code[i] == '`' {
			quote := code[i]
			i++
			for i < len(code) {
				if code[i] == '\\' && quote != '`' {
					i += 2
					if i > len(code) {
						i = len(code)
					}
					continue
				}
				if code[i] == quote {
					i++
					break
				}
				i++
			}
			kind = "string"
		} else if code[i] >= '0' && code[i] <= '9' {
			i++
			for i < len(code) && (isCodeWord(code[i]) || code[i] == '.') {
				i++
			}
			kind = "number"
		} else if isCodeStart(code[i]) {
			i++
			for i < len(code) && isCodeWord(code[i]) {
				i++
			}
			if known[string(code[start:i])] {
				kind = "keyword"
			} else if i < len(code) && code[i] == '(' {
				kind = "function"
			}
		} else if code[i] >= utf8.RuneSelf {
			_, size := utf8.DecodeRuneInString(code[i:])
			i += size
		} else {
			i++
		}
		tokens = append(tokens, CodeToken{Text: code[start:i], Kind: kind})
	}
	return tokens
}

func NormalizeCodeLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if i := strings.IndexAny(language, " -_"); i >= 0 {
		language = language[:i]
	}
	switch language {
	case "javascript", "jsx":
		return "js"
	case "typescript", "tsx":
		return "ts"
	case "golang":
		return "go"
	case "py3":
		return "python"
	case "go", "js", "ts", "python", "py", "sql", "html", "css", "shell", "bash", "sh", "yaml", "yml", "json", "text", "rust", "java", "c", "cpp", "ruby", "php":
		return language
	default:
		return "text"
	}
}

func words(value string) map[string]bool {
	out := make(map[string]bool)
	for _, word := range strings.Fields(value) {
		out[word] = true
	}
	return out
}

func isCodeStart(b byte) bool { return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }
func isCodeWord(b byte) bool  { return isCodeStart(b) || b >= '0' && b <= '9' }
