package ui

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const PublicArticleCSS = `.article-read-tools{display:flex;flex-wrap:wrap;gap:8px;margin:12px 0}.article-read-tools button{min-height:40px;padding:7px 12px;border:1px solid #94a3b8;border-radius:6px;background:#fff;color:#0f172a;font:inherit;cursor:pointer}.article-read-tools button:focus-visible,.article-toc a:focus-visible,.ds-code-block button:focus-visible{outline:3px solid #2563eb;outline-offset:2px}.article-toc{margin:16px 0;padding:16px;border:1px solid #cbd5e1;border-radius:8px;background:#f8fafc;color:#0f172a}.article-toc ul{margin:8px 0;padding-inline-start:22px}.article-toc a{color:inherit}.article-toc a[aria-current=location]{font-weight:700;text-decoration-thickness:2px}.article-reading-body{font-size:calc(1em * var(--article-font-scale,1));scroll-behavior:smooth}.ds-code-block{margin:1.5rem 0;overflow:hidden;border:1px solid #334155;border-radius:10px;background:#0f172a;color:#e2e8f0}.ds-code-block-head{display:flex;align-items:center;gap:.75rem;min-height:42px;padding:0 .75rem;background:#1e293b;color:#cbd5e1;font:500 .8rem/1.3 system-ui,sans-serif}.ds-code-block-language{margin-right:auto}.ds-code-block button{min-height:32px;padding:.35rem .65rem;border:1px solid #64748b;border-radius:6px;background:#334155;color:#f8fafc;font:inherit;cursor:pointer}.ds-code-block pre{margin:0;padding:1rem;overflow:auto;tab-size:4}.ds-code-block code{font:400 .9rem/1.65 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre}.code-token-keyword{color:#c4b5fd}.code-token-string{color:#86efac}.code-token-number{color:#fcd34d}.code-token-comment{color:#94a3b8;font-style:italic}.code-token-function{color:#7dd3fc}.ds-code-copy-status{font-size:.75rem}.ds-code-copy-status:empty{display:none}@media(prefers-color-scheme:dark){.article-read-tools button,.article-toc{background:#111827;color:#f8fafc;border-color:#475569}} @media print{body{background:#fff!important;color:#111!important;font-size:11pt!important}header{display:block!important;padding:0 0 8pt!important;border-bottom:1px solid #aaa!important;background:#fff!important;color:#111!important;box-shadow:none!important}header nav,footer,aside,nav,.article-read-tools,.news-share-box,.news-sidebar-sticky,.nav-toggle,.ds-code-block button{display:none!important}header a{color:#111!important}header svg{max-height:24pt}.news-article-page,.news-article-card,.article-reading-body{display:block!important;width:auto!important;max-width:none!important;margin:0!important;padding:0!important;background:#fff!important;color:#111!important;box-shadow:none!important;border:0!important}.news-article-body{font-size:11pt!important;line-height:1.55!important}.news-article-title{font-size:22pt!important;color:#111!important}a[href^="http"]:after{content:" (" attr(href) ")";font-size:8pt;overflow-wrap:anywhere}}`

const PublicArticleJS = `(function(){document.addEventListener('click',async e=>{const button=e.target.closest('[data-copy-code]');if(!button)return;const figure=button.closest('.ds-code-block');const code=figure?.querySelector('pre code');const status=figure?.querySelector('[data-code-copy-status]');if(!code||!status)return;const value=code.textContent||'';try{if(navigator.clipboard&&window.isSecureContext){await navigator.clipboard.writeText(value)}else{const field=document.createElement('textarea');field.value=value;field.setAttribute('readonly','');field.style.position='fixed';field.style.opacity='0';document.body.append(field);field.select();const copied=document.execCommand('copy');field.remove();if(!copied)throw new Error('copy unavailable')}status.textContent='Kopyalandı';button.textContent='Kopyalandı';window.setTimeout(()=>{status.textContent='';button.textContent='Kodu kopyala'},1800)}catch{status.textContent='Kopyalanamadı';button.textContent='Tekrar deneyin'}});const root=document.querySelector('.article-reading-body');if(!root)return;const key='hydro-article-font-scale';const read=()=>{const v=Number(localStorage.getItem(key));return Number.isFinite(v)&&v>=.9&&v<=1.3?v:1};let scale=read();const apply=()=>{root.style.setProperty('--article-font-scale',String(scale));localStorage.setItem(key,String(scale))};apply();document.addEventListener('click',e=>{const button=e.target.closest('[data-font-size]');if(button){const mode=button.dataset.fontSize;scale=mode==='up'?Math.min(1.3,scale+.1):mode==='down'?Math.max(.9,scale-.1):1;apply()}if(e.target.closest('[data-print-article]'))window.print()});document.addEventListener('click',e=>{const link=e.target.closest('.article-toc a[href^="#"]');if(!link)return;const target=document.getElementById(decodeURIComponent(link.hash.slice(1)));target?.scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'auto':'smooth',block:'start'})});if('IntersectionObserver'in window){const links=[...document.querySelectorAll('.article-toc a[href^="#"]')];const observer=new IntersectionObserver(entries=>entries.forEach(entry=>{if(!entry.isIntersecting)return;links.forEach(link=>{if(link.hash==='#'+entry.target.id)link.setAttribute('aria-current','location');else link.removeAttribute('aria-current')})}),{rootMargin:'-10% 0px -75% 0px'});links.forEach(link=>{const heading=document.getElementById(decodeURIComponent(link.hash.slice(1)));if(heading)observer.observe(heading)})}})();`

// PrepareArticleHTML assigns stable anchor IDs to article H2/H3 headings and
// returns a compact, nested table of contents. Input must already be sanitized.
func PrepareArticleHTML(source string) (string, string, error) {
	body := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(source), body)
	if err != nil {
		return "", "", err
	}
	var entries []tocEntry
	used := make(map[string]int)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "h2" || node.Data == "h3") {
			if title := strings.TrimSpace(nodeText(node)); title != "" {
				id := slugAnchor(title)
				used[id]++
				if used[id] > 1 {
					id += "-" + strconv.Itoa(used[id])
				}
				setAttr(node, "id", id)
				entries = append(entries, tocEntry{ID: id, Title: title, Level: 2})
				if node.Data == "h3" {
					entries[len(entries)-1].Level = 3
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	var rendered bytes.Buffer
	for _, node := range nodes {
		walk(node)
		if err := html.Render(&rendered, node); err != nil {
			return "", "", err
		}
	}
	if len(entries) == 0 {
		return rendered.String(), "", nil
	}
	var toc bytes.Buffer
	if err := articleTOC(nestTOC(entries)).Render(context.Background(), &toc); err != nil {
		return "", "", err
	}
	return rendered.String(), toc.String(), nil
}

// tocEntry is one heading in an article's table of contents; H3 headings
// nest under the H2 before them.
type tocEntry struct {
	ID, Title string
	Level     int
	Children  []tocEntry
}

func nestTOC(entries []tocEntry) []tocEntry {
	var top []tocEntry
	for _, entry := range entries {
		if entry.Level == 3 && len(top) > 0 && top[len(top)-1].Level == 2 {
			parent := &top[len(top)-1]
			parent.Children = append(parent.Children, entry)
			continue
		}
		top = append(top, entry)
	}
	return top
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
