package ui

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/a-h/templ"
)

type PictureProps struct {
	Src, Alt, BlurHash, Class string
	FocalX, FocalY            float64
}

type BlockStyleFieldsProps struct {
	Index               int
	Background, Padding string
}

func blockPresetFieldName(prefix string, index int) string { return prefix + strconv.Itoa(index) }

type MediaFolderItem struct {
	ID, Name, Href string
	Depth          int
}
type MediaMetadataFormProps struct {
	Action, CSRF, ReturnFolder, FolderID, Tags, FocalX, FocalY string
	Folders                                                    []MediaFolderItem
}
type MediaSettingsFormProps struct {
	CSRF                                                                   string
	HotlinkProtection, WatermarkEnabled                                    bool
	AllowedRefererHosts, WatermarkText, WatermarkPosition, WatermarkLogoID string
	WatermarkOpacity, WatermarkMargin                                      string
	WatermarkFolders                                                       []MediaFolderItem
	WatermarkFoldersSelected                                               []string
	WatermarkLogos                                                         []MediaLogoItem
}
type MediaLogoItem struct{ ID, Name string }

func mediaFolderClass(active bool) string {
	if active {
		return "block rounded px-2 py-1 font-medium text-sky-800 bg-sky-50"
	}
	return "block rounded px-2 py-1 text-slate-700 hover:bg-slate-50"
}
func mediaFolderCurrent(selected, id string) string {
	if selected == id {
		return "page"
	}
	return ""
}
func mediaFolderIndent(depth int) string {
	if depth < 0 {
		depth = 0
	}
	if depth > 12 {
		depth = 12
	}
	return fmt.Sprintf("padding-inline-start: %dem", depth+1)
}
func mediaFolderSelected(folders []string, id string) bool {
	for _, folder := range folders {
		if folder == id {
			return true
		}
	}
	return false
}

func PictureVideoEmbed(raw, title string) templ.Component {
	src, ok := safeVideoURL(raw)
	if !ok {
		return templ.ComponentFunc(func(context.Context, io.Writer) error { return nil })
	}
	return videoEmbed(src, title)
}

func pictureStyle(p PictureProps) string {
	return fmt.Sprintf("object-position: %.2f%% %.2f%%; background-color: %s", clampPicture(p.FocalX), clampPicture(p.FocalY), pictureColor(strings.TrimSpace(p.BlurHash)))
}

func safeVideoURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" {
		return "", false
	}
	host, id, base := strings.ToLower(u.Hostname()), "", ""
	switch host {
	case "youtube.com", "www.youtube.com", "m.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		}
		base = "https://www.youtube-nocookie.com/embed/"
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
		base = "https://www.youtube-nocookie.com/embed/"
	case "vimeo.com", "www.vimeo.com":
		id = strings.Trim(u.Path, "/")
		base = "https://player.vimeo.com/video/"
	}
	if id == "" || len(id) > 64 {
		return "", false
	}
	for _, r := range id {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			return "", false
		}
	}
	return base + id, base != ""
}

func pictureColor(hash string) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz#$%*+,-.:;=?@[]^_{|}~"
	if len(hash) < 6 {
		return "#e5e7eb"
	}
	n := 0
	for _, r := range hash[2:6] {
		i := strings.IndexRune(alphabet, r)
		if i < 0 {
			return "#e5e7eb"
		}
		n = n*83 + i
	}
	return fmt.Sprintf("#%02x%02x%02x", (n>>16)&255, (n>>8)&255, n&255)
}

func clampPicture(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 100
	}
	return v * 100
}
