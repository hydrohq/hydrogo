package ui

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

const PublicArticleCSS = `.article-read-tools{display:flex;flex-wrap:wrap;gap:8px;margin:12px 0}.article-read-tools button{min-height:40px;padding:7px 12px;border:1px solid #94a3b8;border-radius:6px;background:#fff;color:#0f172a;font:inherit;cursor:pointer}.article-read-tools button:focus-visible,.article-toc a:focus-visible{outline:3px solid #2563eb;outline-offset:2px}.article-toc{margin:16px 0;padding:16px;border:1px solid #cbd5e1;border-radius:8px;background:#f8fafc;color:#0f172a}.article-toc ul{margin:8px 0;padding-inline-start:22px}.article-toc a{color:inherit}.article-reading-body{font-size:calc(1em * var(--article-font-scale,1));scroll-behavior:smooth} @media(prefers-color-scheme:dark){.article-read-tools button,.article-toc{background:#111827;color:#f8fafc;border-color:#475569}} @media print{body{background:#fff!important;color:#111!important;font-size:11pt!important}header,footer,aside,nav,.article-read-tools,.news-share-box,.news-sidebar-sticky,.nav-toggle{display:none!important}.news-article-page,.news-article-card,.article-reading-body{display:block!important;width:auto!important;max-width:none!important;margin:0!important;padding:0!important;background:#fff!important;color:#111!important;box-shadow:none!important;border:0!important}.news-article-body{font-size:11pt!important;line-height:1.55!important}.news-article-title{font-size:22pt!important;color:#111!important}a[href^="http"]:after{content:" (" attr(href) ")";font-size:8pt;overflow-wrap:anywhere}}`

const PublicArticleJS = `(function(){const root=document.querySelector('.article-reading-body');if(!root)return;const key='hydro-article-font-scale';const read=()=>{const v=Number(localStorage.getItem(key));return Number.isFinite(v)&&v>=.9&&v<=1.3?v:1};let scale=read();const apply=()=>{root.style.setProperty('--article-font-scale',String(scale));localStorage.setItem(key,String(scale))};apply();document.addEventListener('click',e=>{const button=e.target.closest('[data-font-size]');if(button){const mode=button.dataset.fontSize;scale=mode==='up'?Math.min(1.3,scale+.1):mode==='down'?Math.max(.9,scale-.1):1;apply()}if(e.target.closest('[data-print-article]'))window.print()});document.addEventListener('click',e=>{const link=e.target.closest('.article-toc a[href^="#"]');if(!link)return;const target=document.getElementById(decodeURIComponent(link.hash.slice(1)));target?.scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth',block:'start'})})})();`

// PrepareArticleHTML assigns stable anchor IDs to article H2/H3 headings and
// returns a compact, nested table of contents. Input must already be sanitized.
func PrepareArticleHTML(source string) (string, string, error) {
	root, err := html.Parse(strings.NewReader("<div id=\"hydro-article-root\">" + source + "</div>"))
	if err != nil {
		return "", "", err
	}
	var article *html.Node
	var find func(*html.Node)
	find = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" && hasAttr(node, "id", "hydro-article-root") {
			article = node
			return
		}
		for child := node.FirstChild; child != nil && article == nil; child = child.NextSibling {
			find(child)
		}
	}
	find(root)
	if article == nil {
		return "", "", nil
	}
	type entry struct {
		id, title string
		level     int
	}
	var entries []entry
	used := make(map[string]int)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "h2" || node.Data == "h3") {
			level := 2
			if node.Data == "h3" {
				level = 3
			}
			title := strings.TrimSpace(nodeText(node))
			if title != "" {
				id := slugAnchor(title)
				used[id]++
				if used[id] > 1 {
					id += "-" + strconv.Itoa(used[id])
				}
				setAttr(node, "id", id)
				entries = append(entries, entry{id: id, title: title, level: level})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(article)
	var rendered bytes.Buffer
	for child := article.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&rendered, child); err != nil {
			return "", "", err
		}
	}
	if len(entries) == 0 {
		return rendered.String(), "", nil
	}
	var toc strings.Builder
	toc.WriteString(`<nav class="article-toc" aria-label="İçindekiler"><h2>İçindekiler</h2><ul>`)
	inHeading, inSublist := false, false
	for _, item := range entries {
		if item.level == 2 {
			if inSublist {
				toc.WriteString("</ul>")
				inSublist = false
			}
			if inHeading {
				toc.WriteString("</li>")
			}
			toc.WriteString("<li>")
			inHeading = true
		} else if inHeading && !inSublist {
			toc.WriteString("<ul>")
			inSublist = true
		}
		toc.WriteString(`<li><a href="#` + item.id + `">` + html.EscapeString(item.title) + `</a></li>`)
	}
	if inSublist {
		toc.WriteString("</ul>")
	}
	if inHeading {
		toc.WriteString("</li>")
	}
	toc.WriteString("</ul></nav>")
	return rendered.String(), toc.String(), nil
}

func hasAttr(node *html.Node, key, value string) bool {
	for _, attr := range node.Attr {
		if attr.Key == key && attr.Val == value {
			return true
		}
	}
	return false
}

func setAttr(node *html.Node, key, value string) {
	for i := range node.Attr {
		if node.Attr[i].Key == key {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: value})
}

func nodeText(node *html.Node) string {
	var text strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			text.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return text.String()
}

func slugAnchor(value string) string {
	var out strings.Builder
	dash := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dash && out.Len() > 0 {
				out.WriteByte('-')
			}
			out.WriteRune(r)
			dash = false
		} else {
			dash = true
		}
	}
	if out.Len() == 0 {
		return "section"
	}
	return out.String()
}
