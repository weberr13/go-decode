// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decode

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/iancoleman/strcase"
)

// Factory makes Decodeable things described by their kind
type Factory func(kind string) (interface{}, error)

// Factory makes Decodeable things described by their kind
type OneOfFactory func(map[string]interface{}) (interface{}, error)

// PathFactory returns a Factory
type PathFactory func(path string) (func(map[string]interface{}) (interface{}, error), error)

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
func DecodeInto(m map[string]interface{}, o interface{}, pf PathFactory) (interface{}, error) {

	// bail if the passed in object is not a struct
	if reflect.TypeOf(o).Kind() != reflect.Ptr ||
		(reflect.TypeOf(o).Elem().Kind() != reflect.Struct && reflect.TypeOf(o).Elem().Kind() != reflect.Slice) {
		return nil, fmt.Errorf("Target object is not a struct/slice pointer. Unsupprted")
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

		// decode regular fields, recursing into DecodeInto in case of object or array types
		switch v.(type) {
		case map[string]interface{}:
			// Decode a OneOf field and continue if it is
			ok, e := decodeIntoOneOfField(field, fldName, objSchemaName, k, v.(map[string]interface{}), pf)
			if e != nil {
				return nil, e
			}
			if ok {
				continue
			}

			if e := decodeIntoObjectField(field, fldName, v.(map[string]interface{}), pf); e != nil {
				return nil, e
			}
			continue

		case []interface{}:
			if e := decodeIntoArrayField(field, fldName, v.([]interface{}), pf); e != nil {
				return nil, e
			}
			continue

		case []map[string]interface{}:
			if e := decodeIntoArrayOfObjectsField(field, fldName, v.([]map[string]interface{}), pf); e != nil {
				return nil, e
			}
			continue
		}

		// use reflection to set the field
		if field.Kind() == reflect.Ptr {
			vV := reflect.ValueOf(v)
			ft := reflect.TypeOf(field.Interface()).Elem()
			nV := reflect.New(ft)

			if !vV.Type().ConvertibleTo(ft) {
				err := parseAndSetField(fldName, field, nV, vV)
				if err != nil {
					return nil, err
				}
				continue
			}
			nV.Elem().Set(vV.Convert(ft))
			field.Set(nV.Elem().Addr())
			continue
		}

		// special case for empty interfaces - they must represent objects hence we should not be here
		if field.Type().Kind() == reflect.Interface && field.Type().NumMethod() == 0 {
			return nil, fmt.Errorf("Invalid value found for field name %v (expected object, not basic type)\n", fldName)
		}

		if field.CanInterface() {
			newVal := reflect.TypeOf(field.Interface())
			if newVal != reflect.TypeOf(v) {
				if newVal != nil && reflect.TypeOf(v).ConvertibleTo(newVal) {
					field.Set(reflect.ValueOf(v).Convert(newVal))
					continue
				} else {
					return nil, fmt.Errorf("cannot convert value (%v) to field '%s's' type\n", v, fldName)
				}
			}
		}
		field.Set(reflect.ValueOf(v))
	}
	return o, nil
}

type iterator func() (next iterator, obj interface{})

func decodeIntoArray(field reflect.Value, iter iterator, len int, pf PathFactory) error {
	var s reflect.Value
	var ps reflect.Value
	var et reflect.Type
	// three options:
	// - []*Type	- this can be manually created
	// - *[]Type	- this is the codegen option
	// - []Type     - This is a required array
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Slice {
		s = reflect.MakeSlice(field.Type().Elem(), len, len)
		ps = ptr(s)
		et = field.Type().Elem().Elem()
	} else if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Ptr {
		s = reflect.MakeSlice(field.Type(), len, len)
		et = field.Type().Elem().Elem()
	} else if field.Kind() == reflect.Slice {
		s = reflect.MakeSlice(field.Type(), len, len)
		et = field.Type().Elem()
	} else {
		return fmt.Errorf("Invalid field")
	}

	i := 0
	for next, o := iter(); next != nil; next, o = next() {
		pV := reflect.ValueOf(o)

		objm, ok := o.(map[string]interface{})
		if ok {
			pV = reflect.New(et)
			_, err := DecodeInto(objm, pV.Interface(), pf)
			if err != nil {
				return err
			}

			// if field is *[]T, we need to deref this pointer
			if ps.IsValid() {
				pV = pV.Elem()
			}
		}
		s.Index(i).Set(pV)
		i++
	}

	if ps.IsValid() {
		s = ps
	}

	field.Set(s)
	return nil

}

func decodeIntoArrayOfObjectsField(field reflect.Value, fldName string, obj []map[string]interface{}, pf PathFactory) error {
	n := 0
	var i iterator
	i = func() (iterator, interface{}) {
		if n < (len(obj)) {
			n++
			return i, obj[n-1]
		}
		return nil, nil
	}

	return decodeIntoArray(field, i, len(obj), pf)
}

func decodeIntoArrayField(field reflect.Value, fldName string, obj []interface{}, pf PathFactory) error {
	n := 0
	var i iterator
	i = func() (iterator, interface{}) {
		if n < (len(obj)) {
			n++
			return i, obj[n-1]
		}
		return nil, nil
	}

	return decodeIntoArray(field, i, len(obj), pf)
}

func decodeIntoObjectField(field reflect.Value, _ string, v map[string]interface{}, pf PathFactory) error {
	var pV interface{}

	if field.Type().Kind() == reflect.Ptr {
		pV = reflect.New(field.Type().Elem()).Interface()
	} else {
		pV = reflect.New(field.Type()).Interface()
	}

	child, err := DecodeInto(v, pV, pf)
	if err != nil {
		return err
	}
	if field.Kind() != reflect.Ptr {
		field.Set(reflect.ValueOf(child).Elem())
		return nil
	}

	field.Set(reflect.ValueOf(child))
	return nil
}

func decodeIntoOneOfField(field reflect.Value, _ string, objSchemaName string, k string, v map[string]interface{}, pf PathFactory) (bool, error) {
	var pp string
	var f OneOfFactory
	var child interface{}
	var err error

	pp = fmt.Sprintf("%s.%s", objSchemaName, k)

	// get a factory. If factory is nil, but no error, factory was not found for this field
	if f, err = pf(pp); err != nil || f == nil {
		return f != nil, err
	}

	if child, err = f(v); err != nil {
		return false, err
	}

	if child, err = DecodeInto(v, child, pf); err == nil {
		field.Set(reflect.ValueOf(child))
	}
	return err == nil, err
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
func UnmarshalJSONInto(b []byte, o interface{}, pf PathFactory) (interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return DecodeInto(m, o, pf)
}

func parseAndSetField(path string, field, newField, val reflect.Value) error {
	unmarshaler, ok := newField.Interface().(json.Unmarshaler)
	if ok {
		// marshal val back to []byte since it was converted to some underlying type (int/string)
		valBytes, err := json.Marshal(val.Interface());
		if err != nil {
			return fmt.Errorf("Cannot convert value to []byte when attempting to set '%s': %s", path, err) 
		}
		// unmarshal valBytes back into newField object via the unmarshaler
		err = unmarshaler.UnmarshalJSON(valBytes)
		if err != nil {
			return fmt.Errorf("Cannot unmarshal byte values for field '%s': %s", path, err)
		}
		// set field to the value unmarshaled into newField
		field.Set(newField)
		return nil
	}
	return fmt.Errorf("cannot convert value (%v) to field '%s' type", val, path)
}