// Package definitions embeds all YAML resource definition files.
package definitions

import "embed"

// FS contains all embedded resource definition YAML files.
// The embed directive includes all .yaml files under pingone subdirectories.
// Subdirectories (base/, davinci/, etc.) are organizational only.
//
//go:embed pingone/base/*.yaml pingone/davinci/*.yaml
var FS embed.FS
