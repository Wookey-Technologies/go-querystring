// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type flat_1 struct {
	A string
	B int
	C uint
	D float32
	E bool
}

type seen_1 struct {
	D float32 `url:"scale"`
	E bool    `url:"scale,seen"`
}

func TestDecode_types(t *testing.T) {
	//str := "string"
	//strPtr := &str

	tests := []struct {
		in   interface{}
		have url.Values
		want interface{}
	}{
		// Default values from query args
		{
			&flat_1{},
			url.Values{
				"A": {""},
				"B": {"0"},
				"C": {"0"},
				"D": {"0"},
				"E": {"false"},
			},
			&flat_1{},
		},
		// Minor differences
		{
			&flat_1{},
			url.Values{
				"A": {"Johnny"},
				"B": {"3"},
				"C": {"7"},
				"D": {"-2.751e-6"},
				"E": {"1"},
			},
			&flat_1{
				A: "Johnny",
				B: 3,
				C: 7,
				D: -2.751e-6,
				E: true,
			},
		},
		// Names are case sensitive
		{
			&flat_1{},
			url.Values{
				"a": {"Johnny"},
				"b": {"3"},
				"c": {"7"},
				"d": {"-2.751e-6"},
				"e": {"1"},
				"F": {"1"},
				"G": {"1"},
			},
			&flat_1{
			// E: true,
			},
		},
		// New 'seen' option
		{
			&seen_1{},
			url.Values{
				"scale": {"22.7"},
			},
			&seen_1{
				D: 22.7,
				E: true,
			},
		},
	}

	for i, tt := range tests {
		in := reflect.ValueOf(tt.in)
		err := Decode(tt.have, tt.in)
		if err != nil {
			t.Errorf("%d.  Decode(%v, %v) returned error: %v", i, tt.have, in, err)
		}

		if !reflect.DeepEqual(tt.want, tt.in) {
			t.Errorf("%d.  Decode(%v, %v) gave %v, want %v", i, tt.have, in, tt.in, tt.want)
		}
	}
}

type intsize_1 struct {
	B int8
}

func TestDecode_intSize(t *testing.T) {
	values := url.Values{
		"B": {"128"},
	}
	target := intsize_1{}

	if err := Decode(values, &target); err == nil {
		t.Errorf("expected Decode() to return an error on int8 overflow")
	} else if strings.Index(err.Error(), "incompatible") < 0 {
		t.Errorf("expected Decode() to return an error on int8 overflow, got:  %q", err.Error())
	}
}

type uintsize_1 struct {
	B uint16
}

func TestDecode_uintSize(t *testing.T) {
	values := url.Values{
		"B": {"65536"},
	}
	target := uintsize_1{}

	if err := Decode(values, &target); err == nil {
		t.Errorf("expected Decode() to return an error on uint16 overflow")
	} else if strings.Index(err.Error(), "incompatible") < 0 {
		t.Errorf("expected Decode() to return an error on uint16 overflow, got:  %q", err.Error())
	}
}

type complex_1 struct {
	B complex128
}

func TestDecode_complex(t *testing.T) {
	values := url.Values{
		"B": {"65536.282"},
	}
	target := complex_1{}

	if err := Decode(values, &target); err == nil {
		t.Errorf("expected Decode() to return an error on complex128 incompatibility")
	} else if strings.Index(err.Error(), "Unsupported") < 0 {
		t.Errorf("expected Decode() to return an error on complex128 incompatibility, got:  %q", err.Error())
	}
}
