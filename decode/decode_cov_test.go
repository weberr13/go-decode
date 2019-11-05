// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// The test here are meant to call functions that cannot be
// called from outside the package in order to ensure coverage
package decode

import (
	. "github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
	"time"
)

// Test that fields for which there's no value in a payload are still set from defaults specified in struct tags
// *Note* the tests here MUST match the contents of the tags specified in the struct variable up the top of the file
func TestDefaultValues(t *testing.T) {

	Convey("parsing Marshaller field succeeds", t, func() {
		srcv := reflect.ValueOf(&time.Time{})
		srcvp := reflect.PtrTo(srcv.Type())
		pf := reflect.New(srcvp.Elem())

		e := parseAndSetField("Time", pf.Elem(), reflect.ValueOf(&time.Time{}), reflect.ValueOf("2006-01-02T12:34:56Z"))
		So(e, ShouldBeNil)
	})

	Convey("converting Marshaller fields with bad data fails", t, func() {
		ci := make(chan int)
		vum := reflect.ValueOf(&ci)
		srct := time.Time{}
		vumv := reflect.ValueOf(&srct)

		_, e := convertUnmarshallerField("chan", vum, reflect.ValueOf(""))
		So(e, ShouldNotBeNil)
		_, e = convertUnmarshallerField("Time", vumv, vum)
		So(e, ShouldNotBeNil)
		e = parseAndSetField("Time", vumv, vumv, vum)
		So(e, ShouldNotBeNil)
	})

	Convey("attempting to convert unsupported type from string should fail", t, func() {
		ci := make(chan int)
		_, e := convertFromString(reflect.TypeOf(ci), "")
		So(e, ShouldNotBeNil)
	})

	Convey("attempting to use a convertible type as argument to check that type is not supported should fail", t, func() {
		i := 1
		e := unsupportedTypeForDefault(reflect.TypeOf(i))
		So(e, ShouldNotBeNil)
	})
}
