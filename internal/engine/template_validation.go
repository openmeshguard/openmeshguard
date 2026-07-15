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
	if node == nil {
		return nil
	}
	switch typed := node.(type) {
	case *parse.ListNode:
		for _, child := range typed.Nodes {
			if err := validateTemplateNode(child, dotType); err != nil {
				return err
			}
		}
	case *parse.ActionNode:
		return validateTemplateNode(typed.Pipe, dotType)
	case *parse.PipeNode:
		for _, command := range typed.Cmds {
			if err := validateTemplateNode(command, dotType); err != nil {
				return err
			}
		}
	case *parse.CommandNode:
		for _, argument := range typed.Args {
			if err := validateTemplateNode(argument, dotType); err != nil {
				return err
			}
		}
	case *parse.FieldNode:
		_, err := templateSelectorType(dotType, typed.Ident)
		if err != nil {
			return fmt.Errorf("selector %s is invalid: %w", typed.String(), err)
		}
	case *parse.ChainNode:
		if err := validateTemplateNode(typed.Node, dotType); err != nil {
			return err
		}
		if fields, ok := staticTemplateSelector(typed.Node); ok {
			fields = append(fields, typed.Field...)
			if _, err := templateSelectorType(dotType, fields); err != nil {
				return fmt.Errorf("selector %s is invalid: %w", typed.String(), err)
			}
		}
	case *parse.IfNode:
		if err := validateTemplateNode(typed.Pipe, dotType); err != nil {
			return err
		}
		if err := validateTemplateNode(typed.List, dotType); err != nil {
			return err
		}
		return validateTemplateNode(typed.ElseList, dotType)
	case *parse.WithNode:
		if err := validateTemplateNode(typed.Pipe, dotType); err != nil {
			return err
		}
		withType := templatePipelineType(typed.Pipe, dotType)
		if err := validateTemplateNode(typed.List, withType); err != nil {
			return err
		}
		return validateTemplateNode(typed.ElseList, dotType)
	case *parse.RangeNode:
		if err := validateTemplateNode(typed.Pipe, dotType); err != nil {
			return err
		}
		rangeType := templatePipelineType(typed.Pipe, dotType)
		for rangeType != nil && (rangeType.Kind() == reflect.Pointer || rangeType.Kind() == reflect.Interface) {
			rangeType = rangeType.Elem()
		}
		if rangeType != nil && (rangeType.Kind() == reflect.Array || rangeType.Kind() == reflect.Slice || rangeType.Kind() == reflect.Map) {
			rangeType = rangeType.Elem()
		} else {
			rangeType = nil
		}
		if err := validateTemplateNode(typed.List, rangeType); err != nil {
			return err
		}
		return validateTemplateNode(typed.ElseList, dotType)
	case *parse.TemplateNode:
		return validateTemplateNode(typed.Pipe, dotType)
	}
	return nil
}

func templatePipelineType(pipe *parse.PipeNode, dotType reflect.Type) reflect.Type {
	if pipe == nil || len(pipe.Cmds) != 1 || len(pipe.Cmds[0].Args) != 1 {
		return nil
	}
	fields, ok := staticTemplateSelector(pipe.Cmds[0].Args[0])
	if !ok {
		return nil
	}
	fieldType, err := templateSelectorType(dotType, fields)
	if err != nil {
		return nil
	}
	return fieldType
}

func staticTemplateSelector(node parse.Node) ([]string, bool) {
	switch typed := node.(type) {
	case *parse.DotNode:
		return nil, true
	case *parse.FieldNode:
		return append([]string(nil), typed.Ident...), true
	case *parse.ChainNode:
		fields, ok := staticTemplateSelector(typed.Node)
		if !ok {
			return nil, false
		}
		return append(fields, typed.Field...), true
	default:
		return nil, false
	}
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
