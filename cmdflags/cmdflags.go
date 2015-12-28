package cmdflags

import "errors"
import "fmt"
import "strings"
import "reflect"

type argMap map[string]*string
type fieldIndex []int
type fieldMap map[string]fieldIndex

// Allows keys with nil values.
// Disallow multiple values for a single key.
func getArgMap(args []string) (am argMap, err error) {
	key := ""
	am = argMap{}
	for _, arg := range args {
		clean := strings.TrimSpace(arg)
		if strings.HasPrefix(clean, "--") {
			key = clean
			_, hasKey := am[key]
			if !hasKey {
				am[key] = nil
				continue
			}
			err = errors.New("Duplicate keys: " + key)
			return
		}
		if "" == key {
			err = errors.New("Expected key (--), found: " + clean)
			return
		}
		v := am[key]
		if nil != v {
			err = errors.New("Expected single value for: " + key)
		}
		am[key] = &clean // TODO: Investigate, why does this work? Pointer escape analysis?
	}
	return
}

// 1. Add pointer to string support for clean null values
// 2. Figure out how to support more types, specifically url.URL in a clean way.
// 3. Move to package and cleanup.
// 4. Add validation layer
func ParseArgs(src []string, out interface{}) (err error) {
	am, err := getArgMap(src[1:])
	if nil != err {
		return
	}
	fm, err := buildFieldMap(reflect.TypeOf(out).Elem(), "cmd")
	if nil != err {
		return
	}
	rv := reflect.ValueOf(out).Elem()
	for k, v := range am {
		fieldIndex, hasKey := fm[k]
		if !hasKey {
			err = errors.New("Unknown option: " + k)
			return
		}
		field := rv.FieldByIndex(fieldIndex)
		kind := field.Type().Kind()
		if reflect.Bool == kind {
			if nil != v {
				err = errors.New("Syntax error: " + k + " " + *v)
				return
			}
			rv.FieldByIndex(fieldIndex).SetBool(true)
			continue
		}
		if nil == v {
			err = errors.New("Value missing for key: " + k)
		}
		field.Set(reflect.ValueOf(*v))
	}
	return
}

func MakeArgs(o interface{}) (out []string, err error) {
	t := reflect.TypeOf(o)
	v := reflect.ValueOf(o)
	numf := t.NumField()
	for i := 0; i < numf; i++ {
		field := t.Field(i)
		var name string
		name, err = getFieldName(field, "cmd")
		if nil != err {
			return
		}
		kind := field.Type.Kind()
		valField := v.Field(i)
		switch kind {
		case reflect.Bool:
			if false == valField.Bool() {
				continue
			}
			out = append(out, name)
			break
		case reflect.String:
			val := valField.String()
			if "" == val {
				continue
			}
			out = append(out, name, valField.String())
			break
		default:
			err = errors.New("Unsupported type: " + string(kind))
			return out, err
		}
	}
	return out, nil
}

func (m fieldMap) add(t reflect.Type, index []int, tagName string) error {
	numf := t.NumField()
	for i := 0; i < numf; i++ {
		field := t.Field(i)
		fieldIndex := append(index, i)
		if field.Type.Kind() == reflect.Struct && field.Anonymous { // only traverse into embedded structs
			err := m.add(field.Type, fieldIndex, tagName)
			if nil != err {
				return err
			}
			continue
		}
		name, err := getFieldName(field, tagName)
		if nil != err {
			return err
		}
		m[name] = fieldIndex
	}
	return nil
}

func getFieldName(sf reflect.StructField, tagName string) (name string, err error) {
	name = sf.Tag.Get(tagName)
	if "" == name {
		sfstr := fmt.Sprintf("%#v", sf)
		err = errors.New("getArgName failed for: " + sfstr)
		return
	}
	return
}

// Hide in Marshaller/Unmarshaller context avoid rebuilding the map.
func buildFieldMap(t reflect.Type, tagName string) (fm fieldMap, err error) {
	fm = fieldMap{}
	index := []int{}
	err = fm.add(t, index, tagName)
	return fm, err
}
