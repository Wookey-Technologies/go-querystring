// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package query

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
)

var decoderType = reflect.TypeOf(new(Decoder)).Elem()

// Decoder is an interface implemented by any type that wishes to encode
// itself into URL values in a non-standard way.  (Untested extension
// interface, expect problems.)
type Decoder interface {
	DecodeValues(key string, v reflect.Value) error
}

// Decode() aspires to be the inverse function of Encode().
// It takes as input a url.Values object and a pointer to a tagged
// struct and attempts to copy appropriate values from the former
// to the latter guided by 'url' tags on the struct fields.
//
// The current implementation is a *very* partial decoder but
// will serve a majority of use cases.  It doesn't chance pointers
// or embedded structs and only deals with simple builtin data
// (int, uint, float, string) and no slices or maps.  There's
// a notion of a Decoder interface which isn't fully developed
// yet.
//
// Example:
//
//   type BodyAndArgs struct{
//     Name:	     string		`json:"name" xml:"" url:"-"`
//     Description:  string	    `json:"description" xml:"" url:"-"`
//     Style:        string      json:"-" xml:"-" url:"style,omitempty"`
//     StyleSet:     bool        json:"-" xml:"-" url:"style,seen"`
//   }
//
//	 queryArgs := req.URL.Values()
//   baa := BodyAndArgs{}
//
//   if err := Decode(queryArgs, &baa); err != nil {
//      return err
//   }
//   fmt.Printf("Style query arg is:  %q", baa.Style)
//
// The 'toxic' prefix on toxicValues indicates this data is generally
// sourced from untrusted origins and should be handled defensively.
func Decode(toxicValues url.Values, v interface{}) error {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("query:  Decode() expects struct input.  Got %v", val.Kind())
	}

	return absorbValue(toxicValues, val, "")
}

// absorbValue is the approximate inverse of reflectValue and
// oversees conversion of strings in url.Values to struct members
// of a reflection Value.
//
// This is incomplete but some of the scaffolding for a fuller
// implementation of the inverse function is here.  Elaborate
// when needed.
func absorbValue(toxicValues url.Values, val reflect.Value, scope string) error {
	// Not supporting embeddeded structs for now
	// var embedded []reflect.Value

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}

		sv := val.Field(i)
		tag := sf.Tag.Get("url")
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)
		if name == "" {
			if sf.Anonymous && sv.Kind() == reflect.Struct {
				// save embedded struct for later processing
				// embedded = append(embedded, sv)
				continue
			}

			name = sf.Name
		}

		// New 'seen' option.  Always used on a boolean with
		// an explicit 'url'-tagged name, records that a particular
		// name was seen in the url.Values data by setting the
		// bool to true.
		if opts.Contains("seen") {
			if sv.Kind() != reflect.Bool || !sv.CanSet() {
				continue
			}
			if _, ok := toxicValues[name]; ok {
				sv.SetBool(true)
			}
			continue
		}

		if scope != "" {
			name = scope + "[" + name + "]"
		}

		// Use with caution, this hasn't been tested yet.
		if sv.Type().Implements(decoderType) {
			if toxicSlice, ok := toxicValues[name]; ok && len(toxicSlice) > 0 {
				m := sv.Interface().(Decoder)
				if err := m.DecodeValues(toxicSlice[0], sv); err != nil {
					return err
				}
			}
			continue
		}

		// Slice and array processing are doable with a bit of work.
		// Decode generally needs to be more liberal in what it accepts
		// and interpreting separators under 'opts' control is likely
		// the wrong thing to do..
		if sv.Kind() == reflect.Slice || sv.Kind() == reflect.Array {
			// Encode produces strict output, decode wants to be sloppy.
			// Eventually de-decorate 'name' as we go but pass for now.

			// var del byte
			// if opts.Contains("comma") {
			// 	del = ','
			// } else if opts.Contains("space") {
			// 	del = ' '
			// } else if opts.Contains("brackets") {
			// 	name = name + "[]"
			// }

			// if qav, ok := toxicValues.Get(name); ok {
			// 	sv.Add(convertString(qav))
			// }
			continue
		}

		toxicValue := ""
		if toxicSlice, ok := toxicValues[name]; !ok || len(toxicSlice) == 0 {
			continue
		} else {
			toxicValue = toxicSlice[0]
		}

		// Disable the time conversion for now.
		// if sv.Type() == timeType {
		// 	if err := convertString(toxicValue, sv, opts); err != nil {
		// 		return err
		// 	}
		// 	continue
		// }

		for sv.Kind() == reflect.Ptr {
			if sv.IsNil() {
				break
			}
			// sv = sv.Elem()
		}

		// No nested structs for now
		// if sv.Kind() == reflect.Struct {
		// 	absorbValue(toxicValues, sv, name)
		// 	continue
		// }

		if err := convertString(toxicValue, sv, opts); err != nil {
			return err
		}
	}

	// for _, f := range embedded {
	// 	if err := reflectValue(toxicValues, f, scope); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

// convertString attempts to parse the value in toxicStr and use it to
// set the value of the reflected object.
//
// 'toxic' indicates the contents are often from external sources and
// may not be trusted.
func convertString(toxicStr string, v reflect.Value, opts tagOptions) error {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("query:  Attempt to Decode() into nil pointer")
		}
		//v = v.Elem()
	}

	if !v.CanSet() {
		return nil
	}

	// Time not supported for now
	// if v.Type() == timeType {
	// 	t := v.Interface().(time.Time)
	// 	if opts.Contains("unix") {
	// 		return strconv.FormatInt(t.Unix(), 10)
	// 	}
	// 	return t.Format(time.RFC3339)
	// }

	// if v.IsNil() {
	// 	v.Set(reflect.New(v.Type().Elem()))
	// }

	// *NOTE:  decode_test.go keys off of the text 'incompatible' and
	// 'Unsupported' in the errors below.  If you change these error
	// strings, fix those tests.

	switch kind := v.Kind(); kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(toxicStr, 10, 64)
		if err != nil {
			return err
		} else if v.OverflowInt(n) {
			return fmt.Errorf("query:  Query arg value %q incompatible with field value type %v", toxicStr, kind)
		}
		v.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(toxicStr, 10, 64)
		if err != nil {
			return err
		} else if v.OverflowUint(n) {
			return fmt.Errorf("query:  Query arg value %q incompatible with field value type %v", toxicStr, kind)
		}
		v.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(toxicStr, 64)
		if err != nil {
			return err
		} else if v.OverflowFloat(n) {
			return fmt.Errorf("query:  Query arg value %q incompatible with field value type %v", toxicStr, kind)
		}
		v.SetFloat(n)

	case reflect.Bool:
		n, err := strconv.ParseBool(toxicStr)
		if err != nil {
			return err
		}
		v.SetBool(n)

	case reflect.String:
		v.SetString(toxicStr)

	default:
		return fmt.Errorf("query:  Unsupported field value type %v", kind)
	}
	return nil
}
