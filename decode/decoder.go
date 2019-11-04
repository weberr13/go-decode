// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decode

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/iancoleman/strcase"
)

// Factory makes Decodeable things described by their kind
type Factory func(kind string) (interface{}, error)

// Factory makes Decodeable things described by their kind
type OneOfFactory func(map[string]interface{}) (interface{}, error)

// PathFactory returns a Factory
type PathFactory func(path string) (func(map[string]interface{}) (interface{}, error), error)

// DefaultTagName specifies the struct tag used to identify default value for the field
const DefaultTagName = "default"

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
	return UnmarshalJSONIntoWithDefaults(b, o, pf, false)
}

// UnmarshalJSON byte into an instance of object
func UnmarshalJSONIntoWithDefaults(b []byte, o interface{}, pf PathFactory, applyDefaults bool) (interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return decodeInto(m, o, pf, applyDefaults)
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

func DecodeInto(m map[string]interface{}, o interface{}, pf PathFactory) (interface{}, error) {
	return decodeInto(m, o, pf, false)
}

func DecodeIntoWithDefaults(m map[string]interface{}, o interface{}, pf PathFactory, applyDefaults bool) (interface{}, error) {
	return decodeInto(m, o, pf, applyDefaults)
}

// Decode an object's attributes using PathFactory
func decodeInto(m map[string]interface{}, o interface{}, pf PathFactory, applyDefaults bool) (interface{}, error) {
	vo := reflect.ValueOf(o)
	to := vo.Type()
	fm := map[string]reflect.StructField{}

	// bail if the passed in object is not a struct
	if to.Kind() != reflect.Ptr || (to.Elem().Kind() != reflect.Struct && to.Elem().Kind() != reflect.Slice) {
		return nil, fmt.Errorf("Target object is not a struct/slice pointer. Unsupported")
	}

	objSchemaName := to.Elem().Name()

	// only scan struct fields if applyDefaults is specified
	if applyDefaults {
		for i := 0; i < vo.Elem().NumField(); i++ {
			sf := to.Elem().Field(i)
			fm[sf.Name] = sf
		}
	}
	// for each field in the map, if the field is a OneOf (as described in dd), use the associated factory
	for k, v := range m {

		fldName := strcase.ToCamel(k)
		field := vo.Elem().FieldByName(fldName)

		// ignore unknown fields
		if !field.IsValid() {
			continue
		}

		// remove from field set
		delete(fm, fldName)

		// decode regular fields, recursing into decodeInto in case of object or array types
		switch vt := v.(type) {
		case map[string]interface{}:
			// Decode a OneOf field and continue if it is
			ok, e := decodeIntoOneOfField(field, fldName, objSchemaName, k, vt, pf, applyDefaults)
			if e != nil {
				return nil, e
			}
			if !ok {
				if e := decodeIntoObjectField(field, fldName, vt, pf, applyDefaults); e != nil {
					return nil, e
				}
			}

			continue

		case []interface{}:
			if e := decodeIntoArrayField(field, fldName, vt, pf, applyDefaults); e != nil {
				return nil, e
			}
			continue

		case []map[string]interface{}:
			if e := decodeIntoArrayOfObjectsField(field, fldName, vt, pf, applyDefaults); e != nil {
				return nil, e
			}
			continue
		case nil:
			// if field is required, return an error, otherwise ignore it
			if field.Kind() != reflect.Ptr {
				return nil, fmt.Errorf("Invalid value: Null not allowed for required field '%v'\n", fldName)
			}
			continue
		}

		// use reflection to set the field
		if field.Kind() == reflect.Ptr {
			if e := assignPtrField(v, field, fldName); e != nil {
				return nil, e
			}
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

	var err error
	if applyDefaults {
		err = setObjectDefaultValues(fm, vo)
	}

	return o, err
}

func assignPtrField(v interface{}, field reflect.Value, fldName string) error {
	vV := reflect.ValueOf(v)
	ft := reflect.TypeOf(field.Interface()).Elem()
	nV := reflect.New(ft)
	if !vV.Type().ConvertibleTo(ft) {
		return parseAndSetField(fldName, field, nV, vV)
	}
	nV.Elem().Set(vV.Convert(ft))
	field.Set(nV.Elem().Addr())
	return nil
}

type iterator func() (next iterator, obj interface{})

func decodeIntoArray(field reflect.Value, iter iterator, len int, pf PathFactory, applyDefaults bool) error {
	var s reflect.Value
	var ps reflect.Value
	var et reflect.Type
	// three options:
	// - *[]Type	- this is the codegen option
	// - []*Type	- this can be manually created
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
			_, err := decodeInto(objm, pV.Interface(), pf, applyDefaults)
			if err != nil {
				return err
			}
		}

		// If s is a slice of pointers and pV is not a pointer
		if s.Type().Elem().Kind() == reflect.Ptr && pV.Kind() != reflect.Ptr {
			// Init a new zero value of that type in the index of slice, then set with pV value
			s.Index(i).Set(reflect.New(s.Index(i).Type().Elem()))
			s.Index(i).Elem().Set(pV)
			i++
			continue
		}

		//if s is NOT a slice of pointers and pV is a pointer, deref it before calling Set()
		if s.Type().Elem().Kind() != reflect.Ptr && pV.Kind() == reflect.Ptr {
			pV = pV.Elem()
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

func decodeIntoArrayOfObjectsField(field reflect.Value, fldName string, obj []map[string]interface{}, pf PathFactory, applyDefaults bool) error {
	n := 0
	var i iterator
	i = func() (iterator, interface{}) {
		if n < (len(obj)) {
			n++
			return i, obj[n-1]
		}
		return nil, nil
	}

	return decodeIntoArray(field, i, len(obj), pf, applyDefaults)
}

func decodeIntoArrayField(field reflect.Value, fldName string, obj []interface{}, pf PathFactory, applyDefaults bool) error {
	n := 0
	var i iterator
	i = func() (iterator, interface{}) {
		if n < (len(obj)) {
			n++
			return i, obj[n-1]
		}
		return nil, nil
	}

	return decodeIntoArray(field, i, len(obj), pf, applyDefaults)
}

func decodeIntoObjectField(field reflect.Value, _ string, v map[string]interface{}, pf PathFactory, applyDefaults bool) error {
	ft := field.Type()
	if field.Type().Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	pV := reflect.New(ft).Interface()

	child, err := decodeInto(v, pV, pf, applyDefaults)
	if err != nil {
		return err
	}
	cv := reflect.ValueOf(child)
	if field.Kind() != reflect.Ptr {
		cv = cv.Elem()
	}

	field.Set(cv)
	return nil
}

func decodeIntoOneOfField(field reflect.Value, _ string, objSchemaName string, k string, v map[string]interface{}, pf PathFactory, applyDefaults bool) (bool, error) {
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

	if child, err = decodeInto(v, child, pf, applyDefaults); err == nil {
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

func parseAndSetField(path string, field, newField, val reflect.Value) error {
	unmarshaler, ok := newField.Interface().(json.Unmarshaler)
	if ok {
		// marshal val back to []byte since it was converted to some underlying type (int/string)
		valBytes, err := json.Marshal(val.Interface())
		if err != nil {
			return fmt.Errorf("Cannot convert value to []byte when attempting to set '%s': %s\n", path, err)
		}
		// unmarshal valBytes back into newField object via the unmarshaler
		err = unmarshaler.UnmarshalJSON(valBytes)
		if err != nil {
			return fmt.Errorf("Cannot unmarshal byte values for field '%s': %s\n", path, err)
		}
		// set field to the value unmarshaled into newField
		field.Set(newField)
		return nil
	}
	return fmt.Errorf("cannot convert value (%v) to field '%s' type\n", val, path)
}

func setObjectDefaultValues(fm map[string]reflect.StructField, vo reflect.Value) error {
	for k, v := range fm {
		d, ok := v.Tag.Lookup(DefaultTagName)
		if !ok {
			continue
		}
		if err := setFieldDefaultValue(vo, k, d); err != nil {
			return err
		}
	}
	return nil
}

func setFieldDefaultValue(vo reflect.Value, fn, dv string) (err error) {

	f := vo.Elem().FieldByName(fn)
	ft := f.Type()

	if f.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	nV := reflect.New(ft)
	dV := reflect.ValueOf(dv)
	cv := vo

	// Check that the field can be assigned from a default. We are supporting only:
	// 1. types which reflect itself knows how to convert
	// 2. types for which we have actual conversion from string
	// 3. types for which Marshaller is defined and ban be used
	if dV.Type().ConvertibleTo(ft) {
		cv = dV.Convert(ft)
	} else if convertibleFromString(ft) {
		cv, err = convertFromString(ft, dv)
	} else if isUnmarshallableField(f) {
		cv, err = convertUnmarshallerField(fn, f, dV)
	} else if unsupportedTypeForDefault(ft) {
		err = fmt.Errorf("Field is not convertible: %s", fn)
	}
	if err != nil {
		return err
	}
	nV.Elem().Set(cv)

	if f.Kind() != reflect.Ptr {
		nV = nV.Elem()
	}
	f.Set(nV)
	return nil
}

func getUnmarshaller(field reflect.Value) json.Unmarshaler {
	u, ok := field.Interface().(json.Unmarshaler)

	// json.Unmarshal is typically only applicable to pointers, if this is not one, make one
	if !ok && field.Kind() != reflect.Ptr {
		field = ptr(field)
		u, ok = field.Interface().(json.Unmarshaler)
	}

	return u
}

func isUnmarshallableField(field reflect.Value) bool {
	return getUnmarshaller(field) != nil
}

func convertUnmarshallerField(path string, field, val reflect.Value) (vo reflect.Value, err error) {
	// Defensive, since this is only called after a check to isUnmarshallerField.
	u := getUnmarshaller(field)
	if u == nil {
		return vo, fmt.Errorf("cannot convert value (%v) to field '%s' type\n", val, path)
	}
	// marshal val back to []byte since it was converted to some underlying type (int/string)
	vb, err := json.Marshal(val.Interface())
	if err != nil {
		return vo, fmt.Errorf("Cannot convert value to []byte when attempting to set '%s': %s\n", path, err)
	}

	// if the field is a pointer, we need to get to the underlying type
	ft := field.Type()
	if field.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}
	nV := reflect.New(ft)
	u = nV.Interface().(json.Unmarshaler)

	// unmarshal vb back into newField object via the unmarshaler
	err = u.UnmarshalJSON(vb)
	if err != nil {
		return vo, fmt.Errorf("Cannot unmarshal byte values for field '%s': %s\n", path, err)
	}

	// peel off pointer if field is not a pointer
	nV = nV.Elem()
	//if field.Kind() != reflect.Ptr {
	//}

	// set field to the value unmarshaled into newField
	return nV, nil
}

func convertibleFromString(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return true
	}
	return false
}

func unsupportedTypeForDefault(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct, reflect.UnsafePointer:
		return true
	}
	return false
}

func convertFromString(t reflect.Type, v string) (vo reflect.Value, e error) {

	if pc, ok := convMap[int(t.Kind())]; ok {
		return rf(pc.cf(pc.pf(v, pc.sz)))
	}
	return vo, fmt.Errorf("Cannot convert string to unsupported field type: %s(%s)", t.Name(), t.Kind())

}

type pfn func(v string, sz int) (interface{}, error)
type cfn func(interface{}, error) (interface{}, error)
type pc struct {
	sz int
	pf pfn
	cf cfn
}

var pf = func(v string, sz int) (interface{}, error) { return strconv.ParseFloat(v, sz) }
var pi = func(v string, sz int) (interface{}, error) { return strconv.ParseInt(v, 0, sz) }
var pui = func(v string, sz int) (interface{}, error) { return strconv.ParseUint(v, 0, sz) }
var pb = func(v string, _ int) (interface{}, error) { return strconv.ParseBool(v) }
var ci = func(v interface{}, e error) (interface{}, error) { return int(v.(int64)), e }
var ci8 = func(v interface{}, e error) (interface{}, error) { return int8(v.(int64)), e }
var ci16 = func(v interface{}, e error) (interface{}, error) { return int16(v.(int64)), e }
var ci32 = func(v interface{}, e error) (interface{}, error) { return int32(v.(int64)), e }
var ci64 = func(v interface{}, e error) (interface{}, error) { return v, e }
var cui = func(v interface{}, e error) (interface{}, error) { return uint(v.(uint64)), e }
var cui8 = func(v interface{}, e error) (interface{}, error) { return uint8(v.(uint64)), e }
var cui16 = func(v interface{}, e error) (interface{}, error) { return uint16(v.(uint64)), e }
var cui32 = func(v interface{}, e error) (interface{}, error) { return uint32(v.(uint64)), e }
var cui64 = func(v interface{}, e error) (interface{}, error) { return v, e }
var cf32 = func(v interface{}, e error) (interface{}, error) { return float32(v.(float64)), e }
var cf64 = func(v interface{}, e error) (interface{}, error) { return v, e }
var cb = func(v interface{}, e error) (interface{}, error) { return v, e }
var rf = func(v interface{}, e error) (reflect.Value, error) { return reflect.ValueOf(v), e }

var convMap = map[int]pc{
	int(reflect.Int):     pc{0, pi, ci},
	int(reflect.Int8):    pc{8, pi, ci8},
	int(reflect.Int16):   pc{16, pi, ci16},
	int(reflect.Int32):   pc{32, pi, ci32},
	int(reflect.Int64):   pc{64, pi, ci64},
	int(reflect.Uint):    pc{0, pui, cui},
	int(reflect.Uint8):   pc{8, pui, cui8},
	int(reflect.Uint16):  pc{16, pui, cui16},
	int(reflect.Uint32):  pc{32, pui, cui32},
	int(reflect.Uint64):  pc{64, pui, cui64},
	int(reflect.Float32): pc{32, pf, cf32},
	int(reflect.Float64): pc{64, pf, cf64},
	int(reflect.Bool):    pc{0, pb, cb},
}
