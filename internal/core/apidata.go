package core

import "reflect"

// ReadStringField reads a named string field from a struct value via reflection.
// Returns empty string for nil, non-struct, or missing fields.
func ReadStringField(data interface{}, fieldName string) string {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return ""
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return ""
	}
	f := val.FieldByName(fieldName)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

// ReadBoolField reads a named bool field from a struct value via reflection.
// Returns false for nil, non-struct, or missing fields.
func ReadBoolField(data interface{}, fieldName string) bool {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return false
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return false
	}
	f := val.FieldByName(fieldName)
	if !f.IsValid() || f.Kind() != reflect.Bool {
		return false
	}
	return f.Bool()
}

// ReadInterfaceField reads a named interface{} field from a struct via reflection.
// Returns nil for nil, non-struct, missing, or nil-interface fields.
func ReadInterfaceField(data interface{}, fieldName string) interface{} {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}
	f := val.FieldByName(fieldName)
	if !f.IsValid() {
		return nil
	}
	if f.Kind() == reflect.Interface {
		if f.IsNil() {
			return nil
		}
		return f.Interface()
	}
	return f.Interface()
}
