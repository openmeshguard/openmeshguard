package engine

import (
	"fmt"
	"reflect"
	"text/template"
	"text/template/parse"
)

var messageDataType = reflect.TypeOf(messageData{})

func parseValidatedTemplate(name, source string) (*template.Template, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, err
	}
	for _, definition := range tmpl.Templates() {
		if definition.Tree == nil || definition.Root == nil {
			continue
		}
		if err := validateTemplateNode(definition.Root, messageDataType); err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

func validateTemplateNode(node parse.Node, dotType reflect.Type) error {
	return validateTemplateNodeWithVariables(node, dotType, map[string]reflect.Type{"$": messageDataType})
}

func validateTemplateNodeWithVariables(node parse.Node, dotType reflect.Type, variables map[string]reflect.Type) error {
	if node == nil {
		return nil
	}
	switch typed := node.(type) {
	case *parse.ListNode:
		for _, child := range typed.Nodes {
			if err := validateTemplateNodeWithVariables(child, dotType, variables); err != nil {
				return err
			}
		}
	case *parse.ActionNode:
		return validateTemplatePipe(typed.Pipe, dotType, variables)
	case *parse.PipeNode:
		return validateTemplatePipe(typed, dotType, variables)
	case *parse.CommandNode:
		for _, argument := range typed.Args {
			if err := validateTemplateNodeWithVariables(argument, dotType, variables); err != nil {
				return err
			}
		}
	case *parse.FieldNode:
		_, err := templateSelectorType(dotType, typed.Ident)
		if err != nil {
			return fmt.Errorf("selector %s is invalid: %w", typed.String(), err)
		}
	case *parse.VariableNode:
		if _, _, err := templateNodeType(typed, dotType, variables); err != nil {
			return fmt.Errorf("selector %s is invalid: %w", typed.String(), err)
		}
	case *parse.ChainNode:
		if err := validateTemplateNodeWithVariables(typed.Node, dotType, variables); err != nil {
			return err
		}
		if baseType, known, err := templateNodeType(typed.Node, dotType, variables); err != nil {
			return fmt.Errorf("selector %s is invalid: %w", typed.String(), err)
		} else if known {
			if _, err := templateSelectorType(baseType, typed.Field); err != nil {
				return fmt.Errorf("selector %s is invalid: %w", typed.String(), err)
			}
		}
	case *parse.IfNode:
		branchVariables := cloneTemplateVariables(variables)
		if err := validateTemplatePipe(typed.Pipe, dotType, branchVariables); err != nil {
			return err
		}
		if err := validateTemplateNodeWithVariables(typed.List, dotType, cloneTemplateVariables(branchVariables)); err != nil {
			return err
		}
		return validateTemplateNodeWithVariables(typed.ElseList, dotType, cloneTemplateVariables(branchVariables))
	case *parse.WithNode:
		withType := templatePipelineType(typed.Pipe, dotType, variables)
		branchVariables := cloneTemplateVariables(variables)
		if err := validateTemplatePipe(typed.Pipe, dotType, branchVariables); err != nil {
			return err
		}
		if err := validateTemplateNodeWithVariables(typed.List, withType, cloneTemplateVariables(branchVariables)); err != nil {
			return err
		}
		return validateTemplateNodeWithVariables(typed.ElseList, dotType, cloneTemplateVariables(branchVariables))
	case *parse.RangeNode:
		rangeType := templatePipelineType(typed.Pipe, dotType, variables)
		branchVariables := cloneTemplateVariables(variables)
		if err := validateTemplatePipe(typed.Pipe, dotType, branchVariables); err != nil {
			return err
		}
		setRangeVariableTypes(typed.Pipe, rangeType, branchVariables)
		for rangeType != nil && (rangeType.Kind() == reflect.Pointer || rangeType.Kind() == reflect.Interface) {
			rangeType = rangeType.Elem()
		}
		if rangeType != nil && (rangeType.Kind() == reflect.Array || rangeType.Kind() == reflect.Slice || rangeType.Kind() == reflect.Map) {
			rangeType = rangeType.Elem()
		} else {
			rangeType = nil
		}
		if err := validateTemplateNodeWithVariables(typed.List, rangeType, cloneTemplateVariables(branchVariables)); err != nil {
			return err
		}
		return validateTemplateNodeWithVariables(typed.ElseList, dotType, cloneTemplateVariables(branchVariables))
	case *parse.TemplateNode:
		return validateTemplateNodeWithVariables(typed.Pipe, dotType, cloneTemplateVariables(variables))
	}
	return nil
}

func validateTemplatePipe(pipe *parse.PipeNode, dotType reflect.Type, variables map[string]reflect.Type) error {
	if pipe == nil {
		return nil
	}
	for _, command := range pipe.Cmds {
		if err := validateTemplateNodeWithVariables(command, dotType, variables); err != nil {
			return err
		}
	}
	pipelineType := templatePipelineType(pipe, dotType, variables)
	for _, declaration := range pipe.Decl {
		if len(declaration.Ident) > 0 {
			variables[declaration.Ident[0]] = pipelineType
		}
	}
	return nil
}

func templatePipelineType(pipe *parse.PipeNode, dotType reflect.Type, variables map[string]reflect.Type) reflect.Type {
	if pipe == nil || len(pipe.Cmds) != 1 || len(pipe.Cmds[0].Args) != 1 {
		return nil
	}
	fieldType, known, err := templateNodeType(pipe.Cmds[0].Args[0], dotType, variables)
	if err != nil || !known {
		return nil
	}
	return fieldType
}

func templateNodeType(node parse.Node, dotType reflect.Type, variables map[string]reflect.Type) (reflect.Type, bool, error) {
	switch typed := node.(type) {
	case *parse.DotNode:
		return dotType, true, nil
	case *parse.FieldNode:
		fieldType, err := templateSelectorType(dotType, typed.Ident)
		return fieldType, true, err
	case *parse.VariableNode:
		if len(typed.Ident) == 0 {
			return nil, false, nil
		}
		variableType, exists := variables[typed.Ident[0]]
		if !exists {
			return nil, false, nil
		}
		fieldType, err := templateSelectorType(variableType, typed.Ident[1:])
		return fieldType, true, err
	case *parse.ChainNode:
		baseType, known, err := templateNodeType(typed.Node, dotType, variables)
		if err != nil || !known {
			return nil, known, err
		}
		fieldType, err := templateSelectorType(baseType, typed.Field)
		return fieldType, true, err
	case *parse.PipeNode:
		if len(typed.Cmds) != 1 || len(typed.Cmds[0].Args) != 1 {
			return nil, false, nil
		}
		return templateNodeType(typed.Cmds[0].Args[0], dotType, variables)
	default:
		return nil, false, nil
	}
}

func cloneTemplateVariables(input map[string]reflect.Type) map[string]reflect.Type {
	out := make(map[string]reflect.Type, len(input))
	for name, variableType := range input {
		out[name] = variableType
	}
	return out
}

func setRangeVariableTypes(pipe *parse.PipeNode, collectionType reflect.Type, variables map[string]reflect.Type) {
	for collectionType != nil && (collectionType.Kind() == reflect.Pointer || collectionType.Kind() == reflect.Interface) {
		collectionType = collectionType.Elem()
	}
	if pipe == nil || collectionType == nil || len(pipe.Decl) == 0 {
		return
	}
	var keyType, valueType reflect.Type
	switch collectionType.Kind() {
	case reflect.Array, reflect.Slice:
		keyType = reflect.TypeOf(0)
		valueType = collectionType.Elem()
	case reflect.Map:
		keyType = collectionType.Key()
		valueType = collectionType.Elem()
	default:
		return
	}
	if len(pipe.Decl) == 1 {
		variables[pipe.Decl[0].Ident[0]] = valueType
		return
	}
	variables[pipe.Decl[0].Ident[0]] = keyType
	variables[pipe.Decl[1].Ident[0]] = valueType
}

func templateSelectorType(current reflect.Type, fields []string) (reflect.Type, error) {
	for _, field := range fields {
		for current != nil && current.Kind() == reflect.Pointer {
			current = current.Elem()
		}
		if current == nil || current.Kind() == reflect.Interface || current.Kind() == reflect.Map {
			return nil, nil
		}
		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cannot select field %s from %s", field, current)
		}
		structField, found := current.FieldByName(field)
		if !found || !structField.IsExported() {
			return nil, fmt.Errorf("field %s does not exist on %s", field, current)
		}
		current = structField.Type
	}
	return current, nil
}
