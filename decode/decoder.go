// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decode

import (
	"encoding/json"
	"fmt"
	"github.com/iancoleman/strcase"
	"reflect"
)

// Factory makes Decodeable things described by their kind
type Factory func(kind string) (interface{}, error)

type DecoderDesc interface {

	// Create new instance of a type based on its path and discriminator value
	Make(path, dk, dv string) (interface{}, error)

	// return discriminator property for path
	DiscriminatorFor(path string) string
}

// Decode a map into a Decodeable thing given the discriminator and the factory for all possible
// types and embedded types
func Decode(m map[string]interface{}, discriminator string, f Factory) (interface{}, error) {
	kind, ok := m[discriminator].(string)
	if !ok {
		return nil, fmt.Errorf("could not find value for discriminator %s in map %#v", discriminator, m)
	}
	r, err := f(kind)
	if err != nil {
		return nil, err
	}
	for k, v := range m {
		if k == discriminator {
			continue
		}
		obj, ok := v.(map[string]interface{})
		if ok {
			child, err := Decode(obj, discriminator, f)
			if err != nil {
				return nil, err
			}
			reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(reflect.ValueOf(child))
			continue
		}
		if obj, ok := v.([]interface{}); ok {
			elemType := reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Type()
			s := reflect.MakeSlice(elemType, len(obj), len(obj))
			for i := range obj {
				if objm, ok := obj[i].(map[string]interface{}); ok {
					child2, err := Decode(objm, discriminator, f)
					if err != nil {
						return nil, err
					}
					s.Index(i).Set(reflect.Indirect(reflect.ValueOf(child2)))
					continue
				}
				s.Index(i).Set(reflect.ValueOf(obj[i]))
			}
			reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(s)
			continue
		}
		if obj, ok := v.([]map[string]interface{}); ok {
			elemType := reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Type()
			s := reflect.MakeSlice(elemType, len(obj), len(obj))
			for i := range obj {
				child2, err := Decode(obj[i], discriminator, f)
				if err != nil {
					return nil, err
				}
				s.Index(i).Set(reflect.Indirect(reflect.ValueOf(child2)))
			}
			reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(s)
			continue
		}

		if reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Kind() == reflect.Ptr {
			newVal := reflect.TypeOf(reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Interface()).Elem()
			pV := reflect.New(newVal)
			pV.Elem().Set(reflect.ValueOf(v).Convert(newVal))
			reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(pV.Elem().Addr())
			continue
		}
		if reflect.DeepEqual(reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)), reflect.Value{}) {
			fmt.Printf("field by name %v not found", strcase.ToCamel(k))
			continue
		}
		if reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).CanInterface() {
			newVal := reflect.TypeOf(reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Interface())
			if newVal != reflect.TypeOf(v) {
				reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(reflect.ValueOf(v).Convert(newVal))
				continue
			}
		}
		reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(reflect.ValueOf(v))

	}
	return r, nil
}

// Decode an object's attributes using DecoderDesc
func DecodeInto(m map[string]interface{}, o interface{}, dd DecoderDesc) (interface{}, error) {

	// bail if the passed in object is not a struct
	if reflect.TypeOf(o).Kind() != reflect.Ptr || reflect.TypeOf(o).Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("Target object is not a struct pointer. Unsupprted")
	}

	objSchemaName := reflect.TypeOf(o).Elem().Name()

	// for each field in the map, if the field is a OneOf (as described in dd), use the associated factory
	for k, v := range m {

		fldName := strcase.ToCamel(k)
		field := reflect.ValueOf(o).Elem().FieldByName(fldName)

		// ignore unknown fields
		if !field.IsValid() {
			continue
		}

		// Decode a OneOf field and continue if it is
		ok, e := decodeIntoOneOfField(field, fldName, objSchemaName, k, v, dd)
		if e != nil {
			return nil, e
		}
		if ok {
			continue
		}

		// decode regular fields, recursing into DecodeInto in case of object or array types
		switch v.(type) {
		case map[string]interface{}:
			if e := decodeIntoObjectField(field, fldName, v.(map[string]interface{}), dd); e != nil {
				return nil, e
			}
			continue

		case []interface{}:
			if e := decodeIntoArrayField(field, fldName, v.([]interface{}), dd); e != nil {
				return nil, e
			}
			continue

		case []map[string]interface{}:
			if e := decodeIntoArrayOfObjectsField(field, fldName, v.([]map[string]interface{}), dd); e != nil {
				return nil, e
			}
			continue
		}

		// use reflection to set the field
		if field.Kind() == reflect.Ptr {
			newVal := reflect.TypeOf(field.Interface()).Elem()
			pV := reflect.New(newVal)
			pV.Elem().Set(reflect.ValueOf(v).Convert(newVal))
			field.Set(pV.Elem().Addr())
			continue
		}
		if reflect.DeepEqual(field, reflect.Value{}) {
			fmt.Printf("field by name %v not found", fldName)
			continue
		}
		if field.CanInterface() {
			newVal := reflect.TypeOf(field.Interface())
			if newVal != reflect.TypeOf(v) {
				field.Set(reflect.ValueOf(v).Convert(newVal))
				continue
			}
		}
		field.Set(reflect.ValueOf(v))
	}
	return o, nil
}

func decodeIntoArrayOfObjectsField(field reflect.Value, fldName string, obj []map[string]interface{}, dd DecoderDesc) error {
	elemType := field.Type()
	s := reflect.MakeSlice(elemType, len(obj), len(obj))
	for i := range obj {
		child, err := DecodeInto(obj[i], reflect.New(elemType), dd)
		if err != nil {
			return err
		}
		s.Index(i).Set(reflect.ValueOf(child))
	}
	field.Set(s)
	return nil
}

func decodeIntoArrayField(field reflect.Value, name string, obj []interface{}, dd DecoderDesc) error {
	var s reflect.Value
	var ps reflect.Value

	// two options:
	// - []*Type	- this can be manually created
	// - *[]Type	- this is the codegen option
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Slice {
		s = reflect.MakeSlice(field.Type().Elem(), len(obj), len(obj))
		ps = ptr(s)

	} else if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Ptr {
		s = reflect.MakeSlice(field.Type(), len(obj), len(obj))

	}

	// get the underlying element type
	et := field.Type().Elem().Elem()

	for i := range obj {
		objm, ok := obj[i].(map[string]interface{})
		if !ok {
			s.Index(i).Set(reflect.ValueOf(obj[i]))
			continue
		}

		pV := reflect.New(et)
		_, err := DecodeInto(objm, pV.Interface(), dd)
		if err != nil {
			return err
		}

		// if field is *[]T, we need to deref this pointer
		if ps.IsValid() {
			pV = pV.Elem()
		}
		s.Index(i).Set(pV)
	}

	// if we had a *[] type, ps will be initialized, use that as the value to set
	if ps.IsValid() {
		s = ps
	}

	field.Set(s)
	return nil
}

func decodeIntoObjectField(field reflect.Value, fldName string, v map[string]interface{}, dd DecoderDesc) error {
	if field.Kind() != reflect.Ptr {
		return fmt.Errorf("expecting target field %s to be of type object pointer", fldName)
	}
	pV := reflect.New(field.Type().Elem()).Interface()
	child, err := DecodeInto(v, pV, dd)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(child))
	return nil
}

func decodeIntoOneOfField(field reflect.Value, fldName string, objSchemaName string, k string, v interface{}, dd DecoderDesc) (bool, error) {

	var pp string
	var dk string
	var dp interface{}
	var dv string
	var child interface{}
	var err error

	pp = fmt.Sprintf("%s.%s", objSchemaName, k)

	// is this a OneOf property?
	if dk = dd.DiscriminatorFor(pp); dk == "" {
		return false, nil
	}

	obj, ok := v.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("expecting field %s to be of type object", fldName)
	}

	if dp, ok = obj[dk]; !ok {
		return false, fmt.Errorf("expecting OneOf field %s to to have a discriminator property %s", fldName, dk)
	}

	if dv, ok = dp.(string); !ok {
		return false, fmt.Errorf("expecting OneOf field %s's discriminator property `%s` value to be a string", fldName, dk)
	}

	if child, err = dd.Make(pp, dk, dv); err != nil {
		return false, err
	}

	child, err = DecodeInto(obj, child, dd)
	if err != nil {
		return false, err
	}
	field.Set(reflect.ValueOf(child))
	return true, nil
}

func ptr(v reflect.Value) reflect.Value {
	pt := reflect.PtrTo(v.Type()) // create a *T type.
	pv := reflect.New(pt.Elem())  // create a reflect.Value of type *T.
	pv.Elem().Set(v)              // sets pv to point to underlying value of v.
	return pv
}

// UnmarshalJSON byte description of a Decodeable thing
func UnmarshalJSON(b []byte, discriminator string, f Factory) (interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return Decode(m, discriminator, f)
}

// UnmarshalJSON byte into an instance of object
func UnmarshalJSONInto(b []byte, o interface{}, dd DecoderDesc) (interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return DecodeInto(m, o, dd)
}
