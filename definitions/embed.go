// Package definitions embeds all YAML resource definition files.
package definitions

import "embed"

// FS contains all embedded resource definition YAML files.
// The embed directive includes all .yaml files recursively under
// the definitions directory (pingone/davinci/*.yaml, etc.).
//go:embed pingone/davinci/*.yaml
var FS embed.FS
