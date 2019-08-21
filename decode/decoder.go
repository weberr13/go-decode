package decode

import (
	"fmt"
	"reflect"
	"encoding/json"
		
	"github.com/iancoleman/strcase"
)


// Factory makes Decodeable things described by their kind
type Factory func (kind string) (interface{}, error)

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
		if obj, ok := v.([]map[string]interface{}) ; ok {
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
		fmt.Println("field is :", reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Kind())
		if reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Kind() == reflect.Ptr {
			newVal := reflect.TypeOf(reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Interface()).Elem()
			fmt.Printf("%s creating pointer to val %v \n", k, newVal)

			pV := reflect.New(newVal)
			pV.Elem().Set(reflect.ValueOf(v).Convert(newVal))
			reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(pV.Elem().Addr())
			fmt.Println("foo")
			continue
		}
		fmt.Println("field is :", reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Kind())
		reflect.ValueOf(r).Elem().FieldByName(strcase.ToCamel(k)).Set(reflect.ValueOf(v))
		
	}
	return r, nil
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