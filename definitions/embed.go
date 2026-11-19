// Package definitions embeds all YAML resource definition files.
package definitions

import "embed"

// FS contains all embedded resource definition YAML files.
// The embed directive includes all .yaml files under any pingone category
// subdirectory (base/, davinci/, sso/, etc.) — subdirectories are
// organizational only. New category directories do not require editing this
// file.
//
//go:embed pingone/*/*.yaml
var FS embed.FS
