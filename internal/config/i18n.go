package config

import (
	"github.com/OliveiraNt/maned-scout/locales"
	"github.com/invopop/ctxi18n"
)

// InitI18n initializes internationalization settings by loading default localized resources with "en" as the fallback language.
func InitI18n() {
	err := ctxi18n.LoadWithDefault(locales.Content, "en")
	if err != nil {
		panic(err)
	}
}
