// Package locales provide embedded localization resource files for the Maned Scout application.
// It contains YAML files with translated strings for different languages (en, pt-BR)
// that are used for internationalization throughout the application.
package locales

import "embed"

//go:embed en.yaml
//go:embed pt-BR.yaml

// Content is an embedded file system containing localized resource files for initialization and configuration.
var Content embed.FS
