// Package controls embeds the built-in OpenMeshGuard control packs.
package controls

import "embed"

// BuiltinFS contains every built-in pack. Keep this pattern aligned with the
// validation-test glob in internal/engine/controlpack_test.go.
//
//go:embed *.yaml
var BuiltinFS embed.FS
