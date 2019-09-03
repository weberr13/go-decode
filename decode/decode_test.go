// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decode_test

import (
	"encoding/json"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/weberr13/go-decode/decode"
	"testing"
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
	Pets []*BasePet
}

type decoderDesc struct {
}

func (dd decoderDesc) Make(pp, dk, dv string) (interface{}, error) {
	return TypeFactory(fmt.Sprintf("%s(%s=%s)", pp, dk, dv))
}

func (dd decoderDesc) DiscriminatorFor(path string) string {
	disc, ok := DiscriminatedOneOfSchemaMap[path]
	if !ok {
		return ""
	}
	return disc
}

// Bark defines model for Bark.
type Bark struct {
	Type   *string `json:"type,omitempty"`
	Volume *int    `json:"volume,omitempty"`
}

// BasePet defines model for BasePet.
type BasePet struct {
	Age            *int           `json:"age,omitempty"`
	Classes        *[]PetHeritage `json:"classes,omitempty"`
	Heritage       *PetHeritage   `json:"heritage,omitempty"`
	Name           *string        `json:"name,omitempty"`
	NestedHeritage interface{}    `json:"nestedHeritage,omitempty"`
	Species        interface{}    `json:"species,omitempty"`
}

// Cat defines model for Cat.
type Cat struct {
	Action interface{} `json:"action,omitempty"`
	Mood   *string     `json:"mood,omitempty"`
	Type   *string     `json:"type,omitempty"`
}

// Dog defines model for Dog.
type Dog struct {
	Action interface{} `json:"action,omitempty"`
	Kind   *string     `json:"kind,omitempty"`
	Type   *string     `json:"type,omitempty"`
}

// House defines model for House.
type House struct {
	Name  *string `json:"name,omitempty"`
	Rooms *int    `json:"rooms,omitempty"`
	Type  *string `json:"type,omitempty"`
}

// Meow defines model for Meow.
type Meow struct {
	Squeel *string `json:"squeel,omitempty"`
	Type   *string `json:"type,omitempty"`
}

// Palace defines model for Palace.
type Palace struct {
	Halls  *int    `json:"Halls,omitempty"`
	Name   *string `json:"name,omitempty"`
	Towers *int    `json:"towers,omitempty"`
	Type   *string `json:"type,omitempty"`
}

// PetHeritage defines model for PetHeritage.
type PetHeritage struct {
	Class interface{} `json:"class,omitempty"`
	Name  *string     `json:"name,omitempty"`
}

// Purr defines model for Purr.
type Purr struct {
	Heritage *PetHeritage `json:"heritage,omitempty"`
	Type     *string      `json:"type,omitempty"`
}

// Shack defines model for Shack.
type Shack struct {
	Material *string `json:"material,omitempty"`
	Name     *string `json:"name,omitempty"`
	Type     *string `json:"type,omitempty"`
}

func TypeFactory(kind string) (interface{}, error) {
	fm := map[string]func() interface{}{
		"BasePet.heritage.class(type=House)":  NewHouse,
		"BasePet.heritage.class(type=Palace)": NewPalace,
		"BasePet.heritage.class(type=Shack)":  NewShack,
		"BasePet.nestedHeritage(type=House)":  NewHouse,
		"BasePet.nestedHeritage(type=Palace)": NewPalace,
		"BasePet.species(type=Cat)":           NewCat,
		"BasePet.species(type=Dog)":           NewDog,
		"Cat.action(type=Meow)":               NewMeow,
		"Cat.action(type=Purr)":               NewPurr,
		"Dog.action(type=Bark)":               NewBark,
		"PetHeritage.class(type=House)":       NewHouse,
		"PetHeritage.class(type=Palace)":      NewPalace,
		"PetHeritage.class(type=Shack)":       NewShack,
		"Purr.heritage.class(type=House)":     NewHouse,
		"Purr.heritage.class(type=Palace)":    NewPalace,
		"Purr.heritage.class(type=Shack)":     NewShack,
	}
	f, ok := fm[kind]
	if !ok {
		return nil, fmt.Errorf("cannot find type %s", kind)
	}
	return f(), nil
}

// Map <Schema, Discriminator>
var DiscriminatedOneOfSchemaMap = map[string]string{

	"BasePet.heritage.class": "type",
	"BasePet.nestedHeritage": "type",
	"BasePet.species":        "type",
	"Cat.action":             "type",
	"Dog.action":             "type",
	"PetHeritage.class":      "type",
	"Purr.heritage.class":    "type",
}

func NewBark() interface{} {
	_d := "Type"
	return &Bark{Type: &_d}
}

func (r Bark) Discriminator() string {
	return "type"
}

func NewCat() interface{} {
	_d := "Type"
	return &Cat{Type: &_d}
}

func (r Cat) Discriminator() string {
	return "type"
}

func NewDog() interface{} {
	_d := "Type"
	return &Dog{Type: &_d}
}

func (r Dog) Discriminator() string {
	return "type"
}

func NewHouse() interface{} {
	_d := "Type"
	return &House{Type: &_d}
}

func (r House) Discriminator() string {
	return "type"
}

func NewMeow() interface{} {
	_d := "Type"
	return &Meow{Type: &_d}
}

func (r Meow) Discriminator() string {
	return "type"
}

func NewPalace() interface{} {
	_d := "Type"
	return &Palace{Type: &_d}
}

func (r Palace) Discriminator() string {
	return "type"
}

func NewPurr() interface{} {
	_d := "Type"
	return &Purr{Type: &_d}
}

func (r Purr) Discriminator() string {
	return "type"
}

func NewShack() interface{} {
	_d := "Type"
	return &Shack{Type: &_d}
}

func (r Shack) Discriminator() string {
	return "type"
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

var oneOfTestPayload = `
{
    "pets" : [
		{
			"name": "Felix",
			"age": 1,
			"nestedHeritage": {
			  "type": "House",
			  "rooms": 8
			},
			"species": {
				"type": "Cat",
				"kind": "ALOOF",
				"action": {
					"type": "Purr",
					"heritage": {
						"name": "Buckingham",
						"class": {
							"type": "Palace",
							"halls": 7,
							"rooms": 20
						}
					}
				}
			},
			"classes": [
				{
					"name": "Taj Mahal",
					"class": {
						"type": "Palace",
						"halls": 27,
						"rooms": 10
					}
				},
				{
					"name": "White",
					"class": {
						"type": "House",
						"halls": 12,
						"rooms": 100
					}
				}
			],
			"heritage": {
				"name": "Little house in the prairie",
				"class": {
					"type": "House",
					"rooms": 20
				}
			}
		},
		
		{
			"name": "Jerry",
			"age": 1,
			"subtype": {
				"type": "Dog",
				"kind": "SHEPHERD",
				"action": {
					"type": "Bark",
					"volume": 7
				}
			},
			"species": {
				"type": "Dog",
				"kind": "SHEPHERD",
				"action": {
					"type": "Bark",
					"volume": 7
				}
			}
		}
    ]
}
`

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
	Convey("Test OneOf decoding", t, func() {
		v, err := decode.UnmarshalJSONInto([]byte(oneOfTestPayload), &Envelope{}, &decoderDesc{})
		So(err, ShouldBeNil)
		_, err = json.MarshalIndent(v, "", "  ")
		So(err, ShouldBeNil)
	})
}
