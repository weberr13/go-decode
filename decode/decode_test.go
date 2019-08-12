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
	Name string		
}

func (r SubRecord) Kind() string {
	return r.kind
}

func NewSubRecord() decode.Decodeable {
	return &SubRecord{
		kind: "sub_record",
	}
}

type Record struct {
	kind string		 
	Name string		 
	Sub  *SubRecord
}

func (r Record) Kind() string {
	return r.kind
}

func NewRecord() decode.Decodeable {
	return &Record{
		kind: "record",
	}
}

func MyTestFactory(kind string) (decode.Decodeable, error) {
	fm := map[string]func() decode.Decodeable {
		"record": NewRecord,
		"sub_record": NewSubRecord,
	}
	f, ok := fm[kind]
	if !ok {
		return nil, fmt.Errorf("cannot find type %s", kind)
	}
	return f(), nil
}

func TestDecodeNestedObject(t *testing.T) {
	Convey("Decode a nested object", t, func(){
		m := map[string]interface{}{
			"name": "foo",
			"kind": "record",
			"sub": map[string]interface{}{
				"name": "bar",
				"kind": "sub_record",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		So(dec.Kind(), ShouldEqual, "record")
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		So(rec, ShouldResemble, &Record{kind: "record", Name: "foo", Sub: &SubRecord{kind: "sub_record", Name: "bar"}})
	})
	Convey("Unmarshal JSON of a nested object", t, func(){
		m := map[string]interface{}{
			"name": "foo",
			"kind": "record",
			"sub": map[string]interface{}{
				"name": "bar",
				"kind": "sub_record",
			},
		}
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		So(dec.Kind(), ShouldEqual, "record")
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		So(rec, ShouldResemble, &Record{kind: "record", Name: "foo", Sub: &SubRecord{kind: "sub_record", Name: "bar"}})
	})
}