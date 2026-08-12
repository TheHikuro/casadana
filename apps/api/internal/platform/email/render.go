package email

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	texttemplate "text/template"
)

//go:embed templates/email.html templates/email.txt
var templateFS embed.FS

// Parsed once at init: a malformed template is a programming error and should
// blow up at boot, not on the first booking of the season.
var (
	htmlTemplate = htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/email.html"))
	textTemplate = texttemplate.Must(texttemplate.ParseFS(templateFS, "templates/email.txt"))
)

type detail struct {
	Label string
	Value string
}

// content is everything the shared layout needs. Every email in this package
// is the same layout with different content, which is what keeps five
// transactional messages from drifting into five different-looking emails.
type content struct {
	Locale     Locale
	Subject    string
	Preheader  string
	Tagline    string
	Heading    string
	Greeting   string
	Paragraphs []string
	Details    []detail
	Note       string
	SignedName string
	SignedRole string
	ReplyHint  string
}

// render produces the HTML part and the plain-text alternative. Sending both is
// what keeps a transactional email out of the spam folder, and the text part is
// the only thing some clients (and screen readers on them) will show.
func (c content) render() (html, text string, err error) {
	var hb, tb bytes.Buffer
	if err := htmlTemplate.ExecuteTemplate(&hb, "email.html", c); err != nil {
		return "", "", fmt.Errorf("email: render html: %w", err)
	}
	if err := textTemplate.ExecuteTemplate(&tb, "email.txt", c); err != nil {
		return "", "", fmt.Errorf("email: render text: %w", err)
	}
	return hb.String(), tb.String(), nil
}
