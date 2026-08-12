package email

import (
	"sort"
	"strings"
	"testing"
)

// A missing key silently falls back to French, which is exactly the kind of
// half-translated email that reaches a guest without anyone noticing. Comparing
// key sets is what makes that a build failure instead.
func TestCatalog_AllLocalesCarryTheSameKeys(t *testing.T) {
	reference := keysOf(catalog[DefaultLocale])
	for loc, entries := range catalog {
		if loc == DefaultLocale {
			continue
		}
		got := keysOf(entries)
		missing := difference(reference, got)
		extra := difference(got, reference)
		if len(missing) > 0 {
			t.Errorf("locale %s is missing keys: %v", loc, missing)
		}
		if len(extra) > 0 {
			t.Errorf("locale %s has keys %s does not: %v", loc, DefaultLocale, extra)
		}
	}
}

// A key whose value has a different number of %s across locales produces
// "%!s(MISSING)" in one language only — a defect no reviewer reading the French
// copy would ever see.
func TestCatalog_PlaceholderCountsMatchAcrossLocales(t *testing.T) {
	for key, ref := range catalog[DefaultLocale] {
		want := strings.Count(ref, "%")
		for loc, entries := range catalog {
			if got := strings.Count(entries[key], "%"); got != want {
				t.Errorf("key %q: locale %s has %d placeholders, %s has %d",
					key, loc, got, DefaultLocale, want)
			}
		}
	}
}

func TestNormalizeLocale(t *testing.T) {
	tests := map[string]Locale{
		"fr":       LocaleFR,
		"FR":       LocaleFR,
		"fr-FR":    LocaleFR,
		"en":       LocaleEN,
		"en-GB":    LocaleEN,
		"es":       LocaleES,
		"es-419":   LocaleES,
		"":         DefaultLocale,
		"de":       DefaultLocale,
		"  es  ":   LocaleES,
		"nonsense": DefaultLocale,
	}
	for in, want := range tests {
		if got := NormalizeLocale(in); got != want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func difference(a, b []string) []string {
	inB := make(map[string]struct{}, len(b))
	for _, k := range b {
		inB[k] = struct{}{}
	}
	var out []string
	for _, k := range a {
		if _, ok := inB[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}
