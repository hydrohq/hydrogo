package media

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const svgNamespace = "http://www.w3.org/2000/svg"

var (
	ErrInvalidSVG = errors.New("invalid or unsafe SVG document")
	svgElements   = map[string]bool{
		"svg": true, "g": true, "path": true, "circle": true, "ellipse": true,
		"rect": true, "line": true, "polyline": true, "polygon": true,
		"defs": true, "linearGradient": true, "radialGradient": true, "stop": true,
		"title": true, "desc": true, "symbol": true, "use": true, "text": true, "tspan": true,
	}
	svgAttributes = svgAttributeSet()
)

func svgAttributeSet() map[string]bool {
	allowed := strings.Fields("id class xmlns viewBox width height fill fill-opacity fill-rule stroke stroke-width stroke-linecap stroke-linejoin stroke-miterlimit stroke-dasharray stroke-dashoffset stroke-opacity opacity transform d x y x1 y1 x2 y2 cx cy r rx ry points offset gradientUnits gradientTransform spreadMethod stop-color stop-opacity pathLength role aria-label focusable href font-family font-size font-style font-weight text-anchor dominant-baseline dx dy")
	set := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		set[name] = true
	}
	return set
}

// SanitizeSVG copies a bounded SVG subset. It rejects directives and unknown
// elements, strips every non-allowlisted attribute, and accepts local fragment
// references only. The output is XML-escaped by encoding/xml.
func SanitizeSVG(src io.Reader, dst io.Writer, maxBytes int64) error {
	if src == nil || dst == nil || maxBytes <= 0 {
		return ErrInvalidSVG
	}
	data, err := io.ReadAll(io.LimitReader(src, maxBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > maxBytes || !utf8.Valid(data) {
		return ErrInvalidSVG
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = true
	encoder := xml.NewEncoder(&svgLimitWriter{writer: dst, remaining: maxBytes})
	depth, elements := 0, 0
	rootSeen, rootClosed := false, false
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return fmt.Errorf("%w: %v", ErrInvalidSVG, tokenErr)
		}
		switch value := token.(type) {
		case xml.Directive:
			return ErrInvalidSVG
		case xml.ProcInst:
			if !strings.EqualFold(value.Target, "xml") || rootSeen {
				return ErrInvalidSVG
			}
		case xml.StartElement:
			if rootClosed || (value.Name.Space != "" && value.Name.Space != svgNamespace) || !svgElements[value.Name.Local] {
				return ErrInvalidSVG
			}
			if !rootSeen && value.Name.Local != "svg" {
				return ErrInvalidSVG
			}
			if depth == 0 && rootSeen {
				return ErrInvalidSVG
			}
			rootSeen = true
			depth++
			elements++
			if depth > 64 || elements > 10000 {
				return ErrInvalidSVG
			}
			clean := xml.StartElement{Name: xml.Name{Local: value.Name.Local}}
			for _, attr := range value.Attr {
				name := attr.Name.Local
				if name == "xmlns" {
					if attr.Value != svgNamespace {
						return ErrInvalidSVG
					}
					if depth == 1 {
						clean.Attr = append(clean.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: svgNamespace})
					}
					continue
				}
				if attr.Name.Space != "" || !svgAttributes[name] || strings.HasPrefix(strings.ToLower(name), "on") {
					continue
				}
				text := strings.TrimSpace(attr.Value)
				lower := strings.ToLower(text)
				if strings.Contains(lower, "javascript:") || strings.Contains(lower, "data:") || strings.Contains(lower, "http:") || strings.Contains(lower, "https:") || strings.Contains(lower, "//") || strings.Contains(lower, "expression(") {
					return ErrInvalidSVG
				}
				if name == "href" && text != "" && !strings.HasPrefix(text, "#") {
					return ErrInvalidSVG
				}
				clean.Attr = append(clean.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: text})
			}
			if depth == 1 {
				if !hasXMLAttribute(clean.Attr, "xmlns") {
					clean.Attr = append(clean.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: svgNamespace})
				}
				if value.Name.Local != "svg" {
					return ErrInvalidSVG
				}
			}
			if err := encoder.EncodeToken(clean); err != nil {
				return err
			}
		case xml.EndElement:
			if depth <= 0 {
				return ErrInvalidSVG
			}
			if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: value.Name.Local}}); err != nil {
				return err
			}
			depth--
			if depth == 0 {
				rootClosed = true
			}
		case xml.CharData:
			if depth == 0 {
				if strings.TrimSpace(string(value)) != "" {
					return ErrInvalidSVG
				}
				continue
			}
			if err := encoder.EncodeToken(value); err != nil {
				return err
			}
		case xml.Comment:
			// Comments have no rendering value and are intentionally omitted.
		}
	}
	if !rootSeen || !rootClosed || depth != 0 {
		return ErrInvalidSVG
	}
	return encoder.Flush()
}

type svgLimitWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *svgLimitWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.remaining {
		return 0, ErrFileTooLarge
	}
	n, err := w.writer.Write(data)
	w.remaining -= int64(n)
	return n, err
}

func hasXMLAttribute(attrs []xml.Attr, name string) bool {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return true
		}
	}
	return false
}

func sanitizeSVGFile(originalPath, directory string, maxBytes int64) (string, error) {
	source, err := os.Open(originalPath) // #nosec G304 -- path of the temporary upload file the caller created
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.CreateTemp(directory, "svg-sanitized-*")
	if err != nil {
		return "", err
	}
	name := target.Name()
	if err = SanitizeSVG(source, target, maxBytes); err == nil {
		err = target.Sync()
	}
	if closeErr := target.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return filepath.Clean(name), nil
}
