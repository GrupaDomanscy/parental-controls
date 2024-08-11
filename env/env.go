package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

func ReadToCfg(cfg interface{}, fieldMap map[string]string) {
	if reflect.ValueOf(cfg).Kind() != reflect.Pointer {
		panic("ReadToCfg needs a pointer to a struct.")
	}

	if reflect.ValueOf(cfg).IsNil() {
		panic("ReadToCfg needs a non-nil pointer to a struct.")
	}

	if reflect.ValueOf(cfg).Elem().Kind() != reflect.Struct {
		panic("ReadToCfg needs a pointer to a struct.")
	}

	if reflect.ValueOf(cfg).Elem().CanSet() == false {
		panic("ReadToCfg needs a pointer to a struct that can be written.")
	}

	numFields := reflect.ValueOf(cfg).Elem().NumField()

	for i := 0; i < numFields; i++ {
		field := reflect.ValueOf(cfg).Elem().Field(i)
		fieldName := reflect.TypeOf(cfg).Elem().Field(i).Name

		if field.CanSet() == false {
			panic(fmt.Sprintf("Can't assign any value to field %s", fieldName))
		}

		fieldType := field.Kind()

		switch fieldType {
		case reflect.String:
			field.SetString(os.Getenv(fieldMap[fieldName]))
		case reflect.Int:
			intVal, err := strconv.Atoi(os.Getenv(fieldMap[fieldName]))
			if err != nil {
				panic(err)
			}

			field.SetInt(int64(intVal))
		case reflect.Int8:
			intVal, err := strconv.ParseInt(os.Getenv(fieldMap[fieldName]), 10, 8)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Int16:
			intVal, err := strconv.ParseInt(os.Getenv(fieldMap[fieldName]), 10, 16)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Int32:
			intVal, err := strconv.ParseInt(os.Getenv(fieldMap[fieldName]), 10, 32)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Int64:
			intVal, err := strconv.ParseInt(os.Getenv(fieldMap[fieldName]), 10, 64)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Uint:
			uintVal, err := strconv.ParseUint(os.Getenv(fieldMap[fieldName]), 10, 32)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint8:
			uintVal, err := strconv.ParseUint(os.Getenv(fieldMap[fieldName]), 10, 8)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint16:
			uintVal, err := strconv.ParseUint(os.Getenv(fieldMap[fieldName]), 10, 16)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint32:
			uintVal, err := strconv.ParseUint(os.Getenv(fieldMap[fieldName]), 10, 32)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint64:
			uintVal, err := strconv.ParseUint(os.Getenv(fieldMap[fieldName]), 10, 64)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Float32:
			floatVal, err := strconv.ParseFloat(os.Getenv(fieldMap[fieldName]), 32)
			if err != nil {
				panic(err)
			}

			field.SetFloat(floatVal)
		case reflect.Float64:
			floatVal, err := strconv.ParseFloat(os.Getenv(fieldMap[fieldName]), 64)
			if err != nil {
				panic(err)
			}

			field.SetFloat(floatVal)
		default:
			panic("unhandled default case")
		}
	}
}
