package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// extractTypeInternal gets the actual underlying type of field value.
// It will dive into pointers, customTypes and return you the
// underlying value and it's kind.
func (v *validate) extractTypeInternal(current reflect.Value, nullable bool) (reflect.Value, reflect.Kind, bool) {

BEGIN:
	switch current.Kind() {
	case reflect.Ptr:

		nullable = true

		if current.IsNil() {
			return current, reflect.Ptr, nullable
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Interface:

		nullable = true

		if current.IsNil() {
			return current, reflect.Interface, nullable
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Invalid:
		return current, reflect.Invalid, nullable

	default:

		if v.v.hasCustomFuncs {

			if fn, ok := v.v.customFuncs[current.Type()]; ok {
				current = reflect.ValueOf(fn(current))
				goto BEGIN
			}
		}

		return current, current.Kind(), nullable
	}
}

// getStructFieldOKInternal traverses a struct to retrieve a specific field denoted by the provided namespace and
// returns the field, field kind and whether is was successful in retrieving the field at all.
//
// NOTE: when not successful ok will be false, this can happen when a nested struct is nil and so the field
// could not be retrieved because it didn't exist.
func (v *validate) getStructFieldOKInternal(val reflect.Value, namespace string) (current reflect.Value, kind reflect.Kind, nullable bool, found bool) {

BEGIN:
	current, kind, nullable = v.ExtractType(val)
	if kind == reflect.Invalid {
		return
	}

	if namespace == "" {
		found = true
		return
	}

	switch kind {

	case reflect.Ptr, reflect.Interface:
		return

	case reflect.Struct:

		typ := current.Type()
		fld := namespace
		var ns string

		if !typ.ConvertibleTo(timeType) {

			idx := strings.Index(namespace, namespaceSeparator)

			if idx != -1 {
				fld = namespace[:idx]
				ns = namespace[idx+1:]
			} else {
				ns = ""
			}

			bracketIdx := strings.Index(fld, leftBracket)
			if bracketIdx != -1 {
				fld = fld[:bracketIdx]

				ns = namespace[bracketIdx:]
			}

			val = current.FieldByName(fld)
			namespace = ns
			goto BEGIN
		}

	case reflect.Array, reflect.Slice:
		idx := strings.Index(namespace, leftBracket)
		idx2 := strings.Index(namespace, rightBracket)

		arrIdx, _ := strconv.Atoi(namespace[idx+1 : idx2])

		if arrIdx >= current.Len() {
			return
		}

		startIdx := idx2 + 1

		if startIdx < len(namespace) {
			if namespace[startIdx:startIdx+1] == namespaceSeparator {
				startIdx++
			}
		}

		val = current.Index(arrIdx)
		namespace = namespace[startIdx:]
		goto BEGIN

	case reflect.Map:
		idx := strings.Index(namespace, leftBracket) + 1
		idx2 := strings.Index(namespace, rightBracket)

		endIdx := idx2

		if endIdx+1 < len(namespace) {
			if namespace[endIdx+1:endIdx+2] == namespaceSeparator {
				endIdx++
			}
		}

		key := namespace[idx:idx2]

		switch current.Type().Key().Kind() {
		case reflect.Int:
			i, _ := strconv.Atoi(key)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Int8:
			i, _ := strconv.ParseInt(key, 10, 8)
			val = current.MapIndex(reflect.ValueOf(int8(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int16:
			i, _ := strconv.ParseInt(key, 10, 16)
			val = current.MapIndex(reflect.ValueOf(int16(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int32:
			i, _ := strconv.ParseInt(key, 10, 32)
			val = current.MapIndex(reflect.ValueOf(int32(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int64:
			i, _ := strconv.ParseInt(key, 10, 64)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Uint:
			i, _ := strconv.ParseUint(key, 10, 0)
			val = current.MapIndex(reflect.ValueOf(uint(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint8:
			i, _ := strconv.ParseUint(key, 10, 8)
			val = current.MapIndex(reflect.ValueOf(uint8(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint16:
			i, _ := strconv.ParseUint(key, 10, 16)
			val = current.MapIndex(reflect.ValueOf(uint16(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint32:
			i, _ := strconv.ParseUint(key, 10, 32)
			val = current.MapIndex(reflect.ValueOf(uint32(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint64:
			i, _ := strconv.ParseUint(key, 10, 64)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Float32:
			f, _ := strconv.ParseFloat(key, 32)
			val = current.MapIndex(reflect.ValueOf(float32(f)))
			namespace = namespace[endIdx+1:]

		case reflect.Float64:
			f, _ := strconv.ParseFloat(key, 64)
			val = current.MapIndex(reflect.ValueOf(f))
			namespace = namespace[endIdx+1:]

		case reflect.Bool:
			b, _ := strconv.ParseBool(key)
			val = current.MapIndex(reflect.ValueOf(b))
			namespace = namespace[endIdx+1:]

		// reflect.Type = string
		default:
			val = current.MapIndex(reflect.ValueOf(key))
			namespace = namespace[endIdx+1:]
		}

		goto BEGIN
	}

	// if got here there was more namespace, cannot go any deeper
	panic("Invalid field namespace")
}

// asInt returns the parameter as a int64
// or panics if it can't convert
func asInt(param string) int64 {
	i, err := strconv.ParseInt(param, 0, 64)
	panicIf(err)

	return i
}

// asIntFromTimeDuration parses param as time.Duration and returns it as int64
// or panics on error.
func asIntFromTimeDuration(param string) int64 {
	d, err := time.ParseDuration(param)
	if err != nil {
		// attempt parsing as an integer assuming nanosecond precision
		return asInt(param)
	}
	return int64(d)
}

// asIntFromType calls the proper function to parse param as int64,
// given a field's Type t.
func asIntFromType(t reflect.Type, param string) int64 {
	switch t {
	case timeDurationType:
		return asIntFromTimeDuration(param)
	default:
		return asInt(param)
	}
}

// asUint returns the parameter as a uint64
// or panics if it can't convert
func asUint(param string) uint64 {

	i, err := strconv.ParseUint(param, 0, 64)
	panicIf(err)

	return i
}

// asFloat64 returns the parameter as a float64
// or panics if it can't convert
func asFloat64(param string) float64 {
	i, err := strconv.ParseFloat(param, 64)
	panicIf(err)
	return i
}

// asFloat32 returns the parameter as a float32
// or panics if it can't convert
func asFloat32(param string) float64 {
	i, err := strconv.ParseFloat(param, 32)
	panicIf(err)
	return i
}

// asBool returns the parameter as a bool
// or panics if it can't convert
func asBool(param string) bool {

	i, err := strconv.ParseBool(param)
	panicIf(err)

	return i
}

func panicIf(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// Checks if field value matches regex. If fl.Field can be cast to Stringer, it uses the Stringer interfaces
// String() return value. Otherwise, it uses fl.Field's String() value.
func fieldMatchesRegexByStringerValOrString(regexFn func() *regexp.Regexp, fl FieldLevel) bool {
	regex := regexFn()
	switch fl.Field().Kind() {
	case reflect.String:
		return regex.MatchString(fl.Field().String())
	default:
		if stringer, ok := fl.Field().Interface().(fmt.Stringer); ok {
			return regex.MatchString(stringer.String())
		} else {
			return regex.MatchString(fl.Field().String())
		}
	}
}

// mutate validation field value
func mutateWithReflect(from interface{}, to string) error {
	ref := reflect.ValueOf(from)
	return mutate(&ref, to)
}

// mutate recursively validation field value
func mutate(ref *reflect.Value, to string) error {
	switch ref.Kind() {
	case reflect.String:
		ref.Set(reflect.ValueOf(to))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(to, 10, 64)
		if err != nil {
			return err
		}

		switch ref.Kind() {
		case reflect.Int:
			vv := int(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Int8:
			vv := int8(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Int16:
			vv := int16(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Int32:
			vv := int32(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Int64:
			ref.Set(reflect.ValueOf(v))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(to, 10, 64)
		if err != nil {
			return err
		}

		switch ref.Kind() {
		case reflect.Uint:
			vv := uint(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Uint8:
			vv := uint8(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Uint16:
			vv := uint16(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Uint32:
			vv := uint32(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Uint64:
			ref.Set(reflect.ValueOf(v))
		}
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(to, 10)
		if err != nil {
			return err
		}

		switch ref.Kind() {
		case reflect.Float32:
			vv := float32(v)
			ref.Set(reflect.ValueOf(vv))
		case reflect.Float64:
			ref.Set(reflect.ValueOf(v))
		}
	case reflect.Bool:
		v, err := strconv.ParseBool(to)
		if err != nil {
			return err
		}
		ref.Set(reflect.ValueOf(v))
	default:
		return errors.New("unsupported type")
	case reflect.Pointer, reflect.UnsafePointer:
		if !ref.IsNil() {
			p := ref.Elem()
			return mutate(&p, to)
		}

		return interNilType(ref, to)
	}

	return nil
}

func interNilType(ref *reflect.Value, to string) error {
	t := ref.Type()
	val := reflect.New(t.(reflect.Type)).Elem().Interface()
	switch val.(type) {
	case *string:
		ref.Set(reflect.ValueOf(&to))
	case *int, *int8, *int16, *int32, *int64:
		v, err := strconv.ParseInt(to, 10, 64)
		if err != nil {
			return err
		}

		switch val.(type) {
		case *int:
			vv := int(v)
			ref.Set(reflect.ValueOf(&vv))
		case *int8:
			vv := int8(v)
			ref.Set(reflect.ValueOf(&vv))
		case *int16:
			vv := int16(v)
			ref.Set(reflect.ValueOf(&vv))
		case *int32:
			vv := int32(v)
			ref.Set(reflect.ValueOf(&vv))
		case *int64:
			ref.Set(reflect.ValueOf(&v))
		}
	case *uint, *uint8, *uint16, *uint32, *uint64:
		v, err := strconv.ParseUint(to, 10, 64)
		if err != nil {
			return err
		}

		switch val.(type) {
		case *uint:
			vv := uint(v)
			ref.Set(reflect.ValueOf(&vv))
		case *uint8:
			vv := uint8(v)
			ref.Set(reflect.ValueOf(&vv))
		case *uint16:
			vv := uint16(v)
			ref.Set(reflect.ValueOf(&vv))
		case *uint32:
			vv := uint32(v)
			ref.Set(reflect.ValueOf(&vv))
		case *uint64:
			ref.Set(reflect.ValueOf(&v))
		}
	case *float64, *float32:
		v, err := strconv.ParseFloat(to, 64)
		if err != nil {
			return err
		}

		switch val.(type) {
		case *float32:
			vv := float32(v)
			ref.Set(reflect.ValueOf(&vv))
		case *float64:
			ref.Set(reflect.ValueOf(&v))
		}
	case *bool:
		v, err := strconv.ParseBool(to)
		if err != nil {
			return err
		}

		ref.Set(reflect.ValueOf(&v))
	}

	return nil
}
