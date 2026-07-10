package engine

import (
	"fmt"
	"sort"
	"strings"

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
		if found && strings.Contains(path, ".") {
			paths[path] = struct{}{}
		}
		for _, child := range expr.Children() {
			visit(child)
		}
	}
	visit(root)
	if dependencyErr != nil {
		return nil, dependencyErr
	}

	// Keep only the most-specific dependency. A select/index chain contributes
	// each prefix as an AST node, but requires must name the leaf the expression
	// actually reads so a parent map cannot hide an unavailable child.
	for candidate := range paths {
		for other := range paths {
			if candidate != other && strings.HasPrefix(other, candidate+".") {
				delete(paths, candidate)
				break
			}
		}
	}

	var out []string
	for path := range paths {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func dependencyPath(expr celast.NavigableExpr) (string, bool, error) {
	switch expr.Kind() {
	case celast.IdentKind:
		name := contractIdentifier(expr.AsIdent())
		_, root := dependencyRoots[name]
		return name, root, nil
	case celast.SelectKind:
		operand := expr.AsSelect().Operand().(celast.NavigableExpr)
		path, found, err := dependencyPath(operand)
		if err != nil || !found {
			return "", found, err
		}
		return path + "." + expr.AsSelect().FieldName(), true, nil
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
			return path + "." + key, true, nil
		}
		if args[1].Kind() == celast.IdentKind && boundByComprehension(expr, args[1].AsIdent()) {
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

func boundByComprehension(expr celast.NavigableExpr, identifier string) bool {
	current := expr
	for {
		parent, found := current.Parent()
		if !found {
			return false
		}
		if parent.Kind() == celast.ComprehensionKind {
			comprehension := parent.AsComprehension()
			if comprehension.IterVar() == identifier || comprehension.IterVar2() == identifier {
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
