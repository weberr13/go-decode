// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decode_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/weberr13/go-decode/decode"
)

type SubRecord struct {
	kind string
	Name *string
}

type MyString string

func NewSubRecord() interface{} {
	encapsulated := "foo"
	return &SubRecord{
		kind: "sub_record",
		Name: &encapsulated,
	}
}

type SubRecord2 struct {
	kind    string
	Name    MyString
	PtrName *MyString
	Subs    []SubRecord
}

func (r SubRecord2) Discriminator() string {
	return string(r.kind)
}

func NewSubRecord2() interface{} {
	return &SubRecord2{
		kind: "sub_record2",
	}
}

type Record struct {
	kind     string
	Name     string
	Optional *string
	Num      *int
	Slice    []string
	Sub      interface{}
}

func (r Record) Discriminator() string {
	return r.kind
}

func NewRecord() interface{} {
	return &Record{
		kind: "record",
	}
}

type Envelope struct {
	Owners []*PetOwner
}

type LivesInRequiredArray struct {
	Name    string
	LivesIn []string
}

type RequiredBasicTypes struct {
	Age int
	Name string
	Lost bool
}

type LivesInStruct struct {
	LivesIn *RequiredBasicTypes
}

func MyTestFactory(kind string) (interface{}, error) {
	fm := map[string]func() interface{}{
		"record":      NewRecord,
		"sub_record":  NewSubRecord,
		"sub_record2": NewSubRecord2,
	}
	f, ok := fm[kind]
	if !ok {
		return nil, fmt.Errorf("cannot find type %s", kind)
	}
	return f(), nil
}


func TestDecodeNestedObject(t *testing.T) {

	m := map[string]interface{}{
		"name":  "foo",
		"kind":  "record",
		"slice": []string{"foo", "bar"},
		"sub": map[string]interface{}{
			"name": "bar",
			"kind": "sub_record",
		},
	}
	Convey("wrong discriminator, doesn't exist", t, func() {
		_, err := decode.Decode(m, "kib", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("wrong discriminator, not a type", t, func() {
		_, err := decode.Decode(m, "name", MyTestFactory)
		So(err, ShouldNotBeNil)
	})

	// todo: this panics because the expectation is that sub is an object, not base type, but should defend
	//Convey("unrully child object - assigned wrong type", t, func() {
	//	mp := map[string]interface{}{
	//		"name":  "foo",
	//		"kind":  "record",
	//		"slice": []string{"foo", "bar"},
	//		"sub": "12",
	//	}
	//	_, err := decode.Decode(mp, "kind", MyTestFactory)
	//	So(err, ShouldNotBeNil)
	//})
	Convey("unrully child object", t, func() {
		mp := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"name": "bar",
				"kind": "unknown",
			},
		}
		_, err := decode.Decode(mp, "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("decode unruly child object in slice", t, func() {
		mp := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "unknown",
						"name": "1",
					},
				},
				"kind": "sub_record2",
			},
		}
		_, err := decode.Decode(mp, "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("unmarshal unruly child object in slice", t, func() {
		mp := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "unknown",
						"name": "1",
					},
				},
				"kind": "sub_record2",
			},
		}
		b, err := json.Marshal(mp)
		So(err, ShouldBeNil)
		_, err = decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Decode a nested object", t, func() {
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		name := "bar"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub:   &SubRecord{kind: "sub_record", Name: &name},
		})
	})
	Convey("Unmarshal a nested object, different subtype", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind":     "sub_record2",
				"ptr_name": "sub_record2",
				"name":     "sub_record2",
			},
		}
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		encapsulated := MyString("sub_record2")
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind:    "sub_record2",
				PtrName: &encapsulated,
				Name:    encapsulated,
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, different subtype", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind": "sub_record2",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind:    "sub_record2",
				PtrName: nil,
				Name:    "",
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, different subtype, pointer and aliased type values", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind":     "sub_record2",
				"ptr_name": "sub_record2",
				"name":     "sub_record2",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		encapsulated := MyString("sub_record2")
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind:    "sub_record2",
				PtrName: &encapsulated,
				Name:    encapsulated,
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, unexpected/misspelled fields", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind":    "sub_record2",
				"ptrname": "sub_record2",
				"namer":   "sub_record2",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind: "sub_record2",
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Unmarshal JSON of a nested object", t, func() {
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		name := "bar"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub:   &SubRecord{kind: "sub_record", Name: &name},
		})
	})
	Convey("Unmarshal bad JSON", t, func() {
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		_, err = decode.UnmarshalJSON(b[1:], "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - pets1.json", t, func() {
		// load spec from testdata identified by file
		bytes, err := ioutil.ReadFile("testdata/pets1.json")
		So(err, ShouldBeNil)

		v, err := decode.UnmarshalJSONInto(bytes, &Envelope{}, SchemaPathFactory)
		So(err, ShouldBeNil)
		_, err = json.MarshalIndent(v, "", "  ")
		So(err, ShouldBeNil)
	})
	Convey("Test OneOf decoding - array of objects", t, func() {
		b := `{ "name": "john", "owns": [{ "type": "Palace"}, {"type": "House"}]}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldBeNil)
	})
	Convey("Test OneOf decoding - array of user crafted objects ", t, func() {
		var y = struct{ LivesIn *[]struct{ Age *int } }{}
		var x = struct{ LivesIn []*struct{ Age *int } }{}
		var z = struct{ LivesIn []*struct{ Age int } }{}
		m := map[string]interface{}{
			"livesIn": []map[string]interface{}{
				{"age": 7},
			},
		}

		_, err := decode.DecodeInto(m, &y, SchemaPathFactory)
		So(err, ShouldBeNil)
		_, err = decode.DecodeInto(m, &x, SchemaPathFactory)
		So(err, ShouldBeNil)
		_, err = decode.DecodeInto(m, &z, SchemaPathFactory)
		So(err, ShouldBeNil)
	})
	Convey("Test OneOf decoding - array of objects - bad oneOf", t, func() {

		b := `{ "name": "john", "owns": [{ "class": "Palace"}, {"class":12}]}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - array of objects - bad property type", t, func() {
		b := `{ "name": "john", "owns": [{ "class": { "type": "House", "rooms": "string"}}]}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - wrong oneOf discriminator", t, func() {
		b := `{ "name": "john", "livesIn": { "class": "Palace"}}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - bad json", t, func() {
		b := `{ "name": `
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - can decode into object that has required array", t, func() {
		x := LivesInRequiredArray{}
		b := `{ "livesIn": [ "class", "Palace"]}`
		i, err := decode.UnmarshalJSONInto([]byte(b), &x, SchemaPathFactory)
		So(err, ShouldBeNil)
		So(i.(*LivesInRequiredArray), ShouldResemble, &LivesInRequiredArray{LivesIn: []string{"class", "Palace"}})
	})
	Convey("Test OneOf decoding - can decode into object that has required basic types", t, func() {
		y := LivesInStruct{}
		b := `{ "livesIn": { "age": 7, "name": "spot", "lost": false}}`
		i, err := decode.UnmarshalJSONInto([]byte(b), &y, SchemaPathFactory)
		So(err, ShouldBeNil)
		So(i.(*LivesInStruct), ShouldResemble, &LivesInStruct{LivesIn: &RequiredBasicTypes{Age: 7, Name: "spot", Lost: false}})
	})
	Convey("Test OneOf decoding - cannot decode into object is not struct pointer", t, func() {
		b := `{ "name": "john"}`
		i := 1
		_, err := decode.UnmarshalJSONInto([]byte(b), i, SchemaPathFactory)
		So(err, ShouldNotBeNil)
		_, err = decode.UnmarshalJSONInto([]byte(b), &i, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
}
