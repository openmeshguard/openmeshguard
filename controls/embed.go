// Package controls embeds the built-in OpenMeshGuard control packs.
package controls

import "embed"

// BuiltinFS contains every built-in pack and its remediation templates. Keep
// the pack extension here aligned with the validation-test glob in
// internal/engine/controlpack_test.go.
//
//go:embed *.yaml templates/*.tmpl
var BuiltinFS embed.FS
