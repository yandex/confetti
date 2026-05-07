package reflect

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// SetValue converts  given string value to proper target type and sets it
func SetValue(target reflect.Value, value string) error {
	if target.Type().String() == "time.Duration" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		target.SetInt(int64(d))
		return nil
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(value)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		res, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert value '%s' to int: %w", value, err)
		}
		if target.OverflowInt(res) {
			return fmt.Errorf("value %d overflows %s", res, target.Kind())
		}
		target.SetInt(res)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		res, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert value '%s' to uint: %w", value, err)
		}
		if target.OverflowUint(res) {
			return fmt.Errorf("value %d overflows %s", res, target.Kind())
		}
		target.SetUint(res)
	case reflect.Float32, reflect.Float64:
		res, err := strconv.ParseFloat(value, target.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot convert value '%s' to float: %w", value, err)
		}
		target.SetFloat(res)
	case reflect.Bool:
		res, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("cannot convert value '%s' to bool: %w", value, err)
		}
		target.SetBool(res)
	case reflect.Complex64, reflect.Complex128:
		res, err := strconv.ParseComplex(value, target.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot convert value '%s' to complex: %w", value, err)
		}
		target.SetComplex(res)
	}

	return nil
}
