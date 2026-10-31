package translations

import (
	"embed"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	//go:embed errors.*.toml
	files  embed.FS
	bundle = load()
)

// load embeds and parses the authoritative English and Indonesian TOML catalogs, failing initialization on invalid catalog data.
func load() *i18n.Bundle {
	b := i18n.NewBundle(language.English)
	b.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	for _, name := range []string{"errors.en.toml", "errors.id.toml"} {
		data, err := files.ReadFile(name)
		if err != nil {
			panic(err)
		}
		if _, err = b.ParseMessageFileBytes(data, name); err != nil {
			panic(err)
		}
	}
	return b
}

// Message selects a localized error message using language preferences, with a safe fallback for an unknown code.
func Message(locale, code string) string {
	result, err := i18n.NewLocalizer(bundle, locale).Localize(&i18n.LocalizeConfig{MessageID: code})
	if err != nil {
		return "internal error"
	}
	return result
}
