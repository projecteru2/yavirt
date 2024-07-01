package configs

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
)

type checker struct { //nolint
	conf     any
	field    string
	fieldObj reflect.StructField
	val      any
}

func newChecker(conf any, field string) *checker { //nolint
	return &checker{
		conf:  conf,
		field: field,
	}
}

func (c *checker) check() (err error) { //nolint
	if c.conf == nil {
		return errors.Errorf("nil *Config")
	}

	if c.fieldObj, c.val, err = c.getFieldValue(reflect.ValueOf(c.conf), c.field); err != nil {
		return errors.Wrap(err, "")
	}

	if err := c.checkEnum(); err != nil {
		return errors.Wrap(err, "")
	}

	if err := c.checkRange(); err != nil {
		return errors.Wrap(err, "")
	}

	return
}

func (c *checker) checkRange() error { //nolint
	var rang, found = c.fieldObj.Tag.Lookup("range")
	if !found || len(rang) < 1 {
		return nil
	}

	var part = strings.Split(rang, "-")
	if len(part) != 2 {
		return errors.Errorf("invalid range: %s", rang)
	}

	var atoi = func(s []string) ([]int, error) {
		var ar = make([]int, len(s))
		var err error
		for i := 0; i < len(s); i++ {
			ar[i], err = strconv.Atoi(s[i])
			if err != nil {
				return nil, err
			}
		}
		return ar, nil
	}

	var ar, err = atoi(part)
	if err != nil {
		return err
	}

	var min, max = ar[0], ar[1]
	var leng int
	var kind = c.fieldObj.Type.Kind()

	switch kind {
	case reflect.Int:
		leng = c.val.(int) //nolint
	case reflect.String:
		leng = len(c.val.(string))
	default:
		return errors.Errorf("invalid type: %d", kind)
	}

	if leng < min || leng > max {
		return errors.Errorf("invalid length: %d, it should be [%d, %d]", leng, min, max)
	}

	return nil
}

func (c *checker) checkEnum() error { //nolint
	var enum, found = c.fieldObj.Tag.Lookup("enum")
	if !found || len(enum) < 1 {
		return nil
	}

	for _, part := range strings.Split(enum, ",") {
		if fmt.Sprintf("%v", c.val) == strings.TrimSpace(part) {
			return nil
		}
	}

	return errors.Errorf("invalid value: %v", c.val)
}

func (c *checker) getFieldValue(valObj reflect.Value, field string) ( //nolint
	reflect.StructField, any, error) {

	var fieldObj reflect.StructField

	if valObj.Kind() != reflect.Ptr {
		return fieldObj, nil, errors.Errorf("require a ptr")
	}

	var elem = valObj.Elem()
	if !elem.IsValid() {
		return fieldObj, nil, errors.Errorf("invalid value")
	}

	for i := 0; i < elem.NumField(); i++ {
		var name, rest = c.split(field)

		fieldObj = elem.Type().Field(i)
		if !strings.EqualFold(fieldObj.Name, name) {
			continue
		}

		var val = elem.Field(i).Interface()

		if len(rest) < 1 {
			return fieldObj, val, nil
		}

		return c.getFieldValue(reflect.ValueOf(val), rest)
	}

	return fieldObj, nil, errors.Errorf("no such field: %s", field)
}

func (c *checker) split(field string) (string, string) { //nolint
	var i = strings.Index(field, ".")
	if i < 0 {
		return field, ""
	}
	return field[:i], field[i+1:]
}
