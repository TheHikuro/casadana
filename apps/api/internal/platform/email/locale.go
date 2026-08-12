package email

import (
	"strings"
	"time"
)

// Locale is the language a guest-facing email is written in. It mirrors the
// locales the website ships (apps/web/messages): a guest who browsed in
// Spanish should not receive French mail.
type Locale string

const (
	LocaleFR Locale = "fr"
	LocaleEN Locale = "en"
	LocaleES Locale = "es"
)

// DefaultLocale is used when a booking carries no locale (requests created
// before the locale was captured) or an unrecognised one. French is the
// website's source language and the owners' own language.
const DefaultLocale = LocaleFR

// NormalizeLocale maps anything a client might send — "FR", "fr-FR",
// "es-419", "" — onto a locale the catalog actually has copy for.
func NormalizeLocale(s string) Locale {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(s)), "-")
	switch Locale(base) {
	case LocaleFR:
		return LocaleFR
	case LocaleEN:
		return LocaleEN
	case LocaleES:
		return LocaleES
	default:
		return DefaultLocale
	}
}

// formatDate writes a date the way a reader of that locale expects it. Emails
// are read outside any browser, so this cannot be delegated to Intl.
func formatDate(t time.Time, loc Locale) string {
	if loc == LocaleEN {
		return t.Format("Mon 2 Jan 2006")
	}
	return t.Format("02/01/2006")
}
