package config

import (
	"github.com/OliveiraNt/maned-scout/locales"
	"github.com/invopop/ctxi18n"
)

func InitI18n() {
	err := ctxi18n.LoadWithDefault(locales.Content, "en")
	if err != nil {
		panic(err)
	}
}
