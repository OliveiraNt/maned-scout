package locales

import "embed"

//go:embed en.yaml
//go:embed pt-BR.yaml

var Content embed.FS
