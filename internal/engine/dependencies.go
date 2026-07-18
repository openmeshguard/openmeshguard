package engine

import (
	"fmt"
	"sort"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
)

var dependencyRoots = map[string]struct{}{
	"workload":  {},
	"namespace": {},
	"resource":  {},
	"inventory": {},
	"params":    {},
}

func analyzeDependencies(checked *cel.Ast) ([]string, error) {
	root := celast.NavigateAST(checked.NativeRep())
	paths := map[string]struct{}{}
	var dependencyErr error

	var visit func(celast.NavigableExpr)
	visit = func(expr celast.NavigableExpr) {
		if dependencyErr != nil {
			return
		}
		path, found, err := dependencyPath(expr)
		if err != nil {
			dependencyErr = err
			return
		}
		if found {
			segments, pathErr := parseEvidencePath(path)
			if pathErr != nil {
				dependencyErr = pathErr
				return
			}
			if len(segments) > 1 {
				if !isExplicitPathBase(expr) {
					paths[path] = struct{}{}
				}
			} else if !isExplicitPathBase(expr) {
				dependencyErr = fmt.Errorf("dynamic access to %s cannot be represented by an exact dotted requires path", path)
				return
			}
		}
		for _, child := range expr.Children() {
			visit(child)
		}
	}
	visit(root)
	if dependencyErr != nil {
		return nil, dependencyErr
	}

	var out []string
	for path := range paths {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func isExplicitPathBase(expr celast.NavigableExpr) bool {
	parent, found := expr.Parent()
	if !found {
		return false
	}
	switch parent.Kind() {
	case celast.SelectKind:
		return parent.AsSelect().Operand().ID() == expr.ID()
	case celast.CallKind:
		call := parent.AsCall()
		if call.FunctionName() != operators.Index && call.FunctionName() != operators.OptIndex {
			return false
		}
		args := call.Args()
		return len(args) == 2 && args[0].ID() == expr.ID()
	default:
		return false
	}
}

func dependencyPath(expr celast.NavigableExpr) (string, bool, error) {
	switch expr.Kind() {
	case celast.IdentKind:
		name := contractIdentifier(expr.AsIdent())
		if identifierBoundByComprehension(expr, expr.AsIdent()) {
			return "", false, nil
		}
		_, root := dependencyRoots[name]
		return name, root, nil
	case celast.SelectKind:
		operand := expr.AsSelect().Operand().(celast.NavigableExpr)
		path, found, err := dependencyPath(operand)
		if err != nil || !found {
			return "", found, err
		}
		return appendEvidenceSegment(path, expr.AsSelect().FieldName()), true, nil
	case celast.CallKind:
		call := expr.AsCall()
		if call.FunctionName() != operators.Index && call.FunctionName() != operators.OptIndex {
			return "", false, nil
		}
		args := call.Args()
		if len(args) != 2 {
			return "", false, nil
		}
		base := args[0].(celast.NavigableExpr)
		path, found, err := dependencyPath(base)
		if err != nil || !found {
			return "", found, err
		}
		if key, ok := stringLiteral(args[1]); ok {
			return appendEvidenceSegment(path, key), true, nil
		}
		if args[1].Kind() == celast.IdentKind && identifierBoundByComprehension(args[1].(celast.NavigableExpr), args[1].AsIdent()) {
			return path, true, nil
		}
		return "", false, fmt.Errorf("dynamic index into %s cannot be represented by a dotted requires path", path)
	default:
		return "", false, nil
	}
}

func stringLiteral(expr celast.Expr) (string, bool) {
	if expr.Kind() != celast.LiteralKind {
		return "", false
	}
	value, ok := expr.AsLiteral().Value().(string)
	return value, ok
}

func identifierBoundByComprehension(expr celast.NavigableExpr, identifier string) bool {
	current := expr
	for {
		parent, found := current.Parent()
		if !found {
			return false
		}
		if parent.Kind() == celast.ComprehensionKind {
			comprehension := parent.AsComprehension()
			if comprehension.IterRange().ID() != current.ID() &&
				(comprehension.IterVar() == identifier || comprehension.IterVar2() == identifier || comprehension.AccuVar() == identifier) {
				return true
			}
		}
		current = parent
	}
}

func contractIdentifier(identifier string) string {
	if identifier == namespaceCELVariable {
		return "namespace"
	}
	return identifier
}

func knownBooleanDependency(path string) bool {
	switch path {
	case "workload.mtls.clientTLSContradiction",
		"workload.authorization.broadAllow",
		"workload.verified.plaintextObserved",
		"inventory.multiCluster.participationDetected",
		"inventory.multiCluster.evaluated",
		"inventory.dataPlane.ztunnel.present",
		"resource.isPubliclyExposed":
		return true
	default:
		return false
	}
}

func directKnownBooleanExpression(checked *cel.Ast) bool {
	root := celast.NavigateAST(checked.NativeRep())
	path, found, err := dependencyPath(root)
	return err == nil && found && knownBooleanDependency(path)
}

// AffectedControlIDs derives permission impact from the controls loaded for the
// current scan. Paths identify evidence that may be unavailable; scopes identify
// target sets that may be incomplete because their backing resources could not
// be listed.
func AffectedControlIDs(packs []Pack, paths, scopes []string) []string {
	scopeSet := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scopeSet[scope] = struct{}{}
	}
	ids := map[string]struct{}{}
	for _, pack := range packs {
		for _, control := range pack.Controls {
			if _, affected := scopeSet[control.Scope]; affected {
				ids[control.ID] = struct{}{}
				continue
			}
			dependencies := append([]string(nil), requiredPaths(control)...)
			dependencies = append(dependencies, control.applicabilityPaths...)
			dependencies = append(dependencies, control.expressionPaths...)
			for _, dependency := range dependencies {
				for _, path := range paths {
					if evidencePathsOverlap(dependency, path) {
						ids[control.ID] = struct{}{}
						break
					}
				}
			}
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
