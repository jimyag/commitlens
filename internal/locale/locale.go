// Package locale provides English / Chinese UI strings for the interactive TUI.
// Cmd prints (config errors, stats warnings, web URL) are always English.
package locale

import (
	"os"
	"strings"
)

// Tag is a supported UI language.
type Tag string

const (
	En Tag = "en"
	Zh Tag = "zh"
)

var current Tag = En

// Init resolves language from config, then COMMITLENS_LANG, then LANG/LC_MESSAGES, defaulting to English.
func Init(configLang string) {
	Set(ResolveTag(
		strings.TrimSpace(configLang),
		os.Getenv("COMMITLENS_LANG"),
		os.Getenv("LC_MESSAGES"),
		os.Getenv("LANG"),
	))
}

// Set switches active locale. Unknown values fall back to En.
func Set(tag Tag) {
	switch tag {
	case Zh:
		current = Zh
	default:
		current = En
	}
}

// Current returns the active tag.
func Current() Tag { return current }

// T returns the localized string for key, with fallback to English then key.
func T(key string) string {
	if s := bundles[current][key]; s != "" {
		return s
	}
	if s := bundles[En][key]; s != "" {
		return s
	}
	return key
}

// ResolveTag picks the first valid language from prefs, then detects zh* from LANG-style strings, else en.
func ResolveTag(prefs ...string) Tag {
	for _, p := range prefs {
		if t := parseTag(p); t == En || t == Zh {
			return t
		}
	}
	for _, p := range prefs {
		lo := strings.ToLower(p)
		if strings.Contains(lo, "zh") {
			return Zh
		}
	}
	return En
}

func parseTag(s string) Tag {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case "en", "english":
		return En
	case "zh", "zh-cn", "zh-hans", "chinese", "cmn-hans":
		return Zh
	}
	if strings.HasPrefix(s, "zh-") {
		return Zh
	}
	if strings.HasPrefix(s, "en-") {
		return En
	}
	return ""
}
