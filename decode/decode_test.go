package decode_test

import (
	"testing"
	"fmt"
	"encoding/json"

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
	kind *MyString		
	Name MyString
	Subs []SubRecord	
}

func (r SubRecord2) Discriminator() string {
	return string(*r.kind)
}

func NewSubRecord2() interface{} {
	encapsulated := MyString("sub_record2")
	return &SubRecord2{
		kind: &encapsulated,
	}
}

type Record struct {
	kind string		 
	Name string	 
	Optional *string
	Num *int
	Slice []string
	Sub  interface{}
}

func (r Record) Discriminator() string {
	return r.kind
}

func NewRecord() interface{} {
	return &Record{
		kind: "record",
	}
}

func MyTestFactory(kind string) (interface{}, error) {
	fm := map[string]func() interface{} {
		"record": NewRecord,
		"sub_record": NewSubRecord,
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
		"name": "foo",
		"kind": "record",
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
	Convey("unrully child object", t, func() {
		mp := map[string]interface{}{
			"name": "foo",
			"kind": "record",
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
			"name": "foo",
			"kind": "record",
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
			"name": "foo",
			"kind": "record",
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
	Convey("Decode a nested object", t, func(){
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		name := "bar"
		So(rec, ShouldResemble, &Record{
			kind: "record", 
			Name: "foo", 
			Slice: []string{"foo", "bar"}, 
			Sub: &SubRecord{kind: "sub_record", Name: &name},
			})
	})	
	Convey("Unmarshal a nested object, different subtype", t, func(){
		m := map[string]interface{}{
			"name": "foo",
			"kind": "record",
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
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		encapsulated := MyString("sub_record2")
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind: "record", 
			Name: "foo", 
			Slice: []string{"foo", "bar"}, 
			Sub: &SubRecord2{
				kind: &encapsulated, 
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, different subtype", t, func(){
		m := map[string]interface{}{
			"name": "foo",
			"kind": "record",
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
		encapsulated := MyString("sub_record2")
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind: "record", 
			Name: "foo", 
			Slice: []string{"foo", "bar"}, 
			Sub: &SubRecord2{
				kind: &encapsulated, 
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Unmarshal JSON of a nested object", t, func(){
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		name := "bar"
		So(rec, ShouldResemble, &Record{
			kind: "record", 
			Name: "foo", 
			Slice: []string{"foo", "bar"}, 
			Sub: &SubRecord{kind: "sub_record", Name: &name},
			})
	})
	Convey("Unmarshal bad JSON", t, func(){
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		_, err = decode.UnmarshalJSON(b[1:], "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
}