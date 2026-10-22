package mailer

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/url"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
)

var emailColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var emailAssetIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

type EmailBranding struct {
	LegalName, PhysicalAddress, PrivacyPolicyURL        string
	FooterNotice, LogoAssetID, PrimaryColor, FooterText string
}

func (b EmailBranding) Normalize() (EmailBranding, error) {
	if len(b.LegalName) > 256 || len(b.PhysicalAddress) > 512 || len(b.PrivacyPolicyURL) > 2048 || len(b.FooterNotice) > 1024 || len(b.FooterText) > 1024 {
		return EmailBranding{}, errors.New("email branding text is too long")
	}
	if strings.ContainsAny(b.LegalName+b.PhysicalAddress+b.PrivacyPolicyURL+b.FooterNotice+b.FooterText+b.LogoAssetID+b.PrimaryColor, "\r\n\x00") {
		return EmailBranding{}, errors.New("email branding contains invalid characters")
	}
	b.LegalName = strings.TrimSpace(b.LegalName)
	b.PhysicalAddress = strings.TrimSpace(b.PhysicalAddress)
	b.PrivacyPolicyURL = strings.TrimSpace(b.PrivacyPolicyURL)
	b.FooterNotice = strings.TrimSpace(b.FooterNotice)
	b.FooterText = strings.TrimSpace(b.FooterText)
	b.LogoAssetID = strings.TrimSpace(b.LogoAssetID)
	b.PrimaryColor = strings.TrimSpace(b.PrimaryColor)
	if b.PrimaryColor == "" {
		b.PrimaryColor = "#2563eb"
	}
	if !emailColorPattern.MatchString(b.PrimaryColor) {
		return EmailBranding{}, errors.New("email accent color must be a six digit hex color")
	}
	if b.LogoAssetID != "" && !emailAssetIDPattern.MatchString(b.LogoAssetID) {
		return EmailBranding{}, errors.New("email logo asset is invalid")
	}
	if b.PrivacyPolicyURL != "" {
		u, err := url.ParseRequestURI(b.PrivacyPolicyURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return EmailBranding{}, errors.New("privacy policy link must be an absolute web URL")
		}
	}
	return b, nil
}

type EmailBrandingProvider interface {
	ReadEmailBranding(context.Context) (EmailBranding, error)
}

// BrandedSender reads settings at delivery time, so changes apply to the next
// direct send and the next delivery from the existing mail spool.
type BrandedSender struct {
	Next         Sender
	Settings     EmailBrandingProvider
	PublicOrigin string
}

func (s BrandedSender) Send(ctx context.Context, message Message) error {
	if s.Next == nil {
		return ErrSenderRequired
	}
	if s.Settings != nil {
		branding, err := s.Settings.ReadEmailBranding(ctx)
		if err != nil {
			return err
		}
		branding, err = branding.Normalize()
		if err != nil {
			return err
		}
		message = applyEmailBranding(message, branding, s.PublicOrigin)
	}
	return s.Next.Send(ctx, message)
}

func applyEmailBranding(message Message, branding EmailBranding, origin string) Message {
	baseText := message.Body
	parts := make([]string, 0, 5)
	for _, value := range []string{branding.LegalName, branding.PhysicalAddress, branding.FooterText, branding.FooterNotice, branding.PrivacyPolicyURL} {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 && branding.LogoAssetID == "" && branding.PrimaryColor == "#2563eb" {
		return message
	}
	if len(parts) > 0 {
		if message.Body != "" {
			message.Body = strings.TrimRight(message.Body, "\r\n") + "\n\n" + strings.Join(parts, "\n")
		} else {
			message.Body = strings.Join(parts, "\n")
		}
	}
	content := message.HTMLBody
	if strings.TrimSpace(content) == "" {
		content = renderEmailTemplate("plain", baseText)
	}
	message.HTMLBody = brandedHTML(content, branding, origin)
	return message
}

// brandedHTML adds the branding header and footer (templates.go) inside
// the body of an HTML message, leaving the message itself untouched.
func brandedHTML(content string, branding EmailBranding, origin string) string {
	logoURL := ""
	if branding.LogoAssetID != "" && strings.TrimSpace(origin) != "" {
		logoURL = strings.TrimRight(origin, "/") + "/media/" + url.PathEscape(branding.LogoAssetID)
	}
	name := branding.LegalName
	if name == "" {
		name = "HydroGo"
	}
	color := branding.PrimaryColor
	if !emailColorPattern.MatchString(color) {
		color = "#2563eb" // the value reaches an inline style, so never trust it unchecked
	}
	header := renderEmailTemplate("header", emailHeaderData{LogoURL: logoURL, LogoAlt: branding.LegalName, Name: name, Color: template.CSS(color)}) // #nosec G203 -- color matched emailColorPattern (six-digit hex) just above
	lines := make([]string, 0, 4)
	for _, value := range []string{branding.LegalName, branding.PhysicalAddress, branding.FooterText, branding.FooterNotice} {
		if value = strings.TrimSpace(value); value != "" {
			lines = append(lines, value)
		}
	}
	footer := renderEmailTemplate("footer", emailFooterData{Lines: lines, PrivacyPolicyURL: branding.PrivacyPolicyURL})
	doc, err := xhtml.Parse(strings.NewReader(content))
	if err != nil {
		return content
	}
	body := findEmailNode(doc, "body")
	if body == nil {
		return content
	}
	prependEmailFragment(body, header)
	appendEmailFragment(body, footer)
	var output bytes.Buffer
	if err := xhtml.Render(&output, doc); err != nil {
		return content
	}
	return output.String()
}

func findEmailNode(node *xhtml.Node, name string) *xhtml.Node {
	if node.Type == xhtml.ElementNode && node.Data == name {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findEmailNode(child, name); found != nil {
			return found
		}
	}
	return nil
}

func prependEmailFragment(parent *xhtml.Node, markup string) {
	for _, node := range emailFragment(parent, markup) {
		parent.InsertBefore(node, parent.FirstChild)
	}
}

func appendEmailFragment(parent *xhtml.Node, markup string) {
	for _, node := range emailFragment(parent, markup) {
		parent.AppendChild(node)
	}
}

func emailFragment(parent *xhtml.Node, markup string) []*xhtml.Node {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: parent.DataAtom}
	nodes, _ := xhtml.ParseFragment(strings.NewReader(markup), contextNode)
	return nodes
}
