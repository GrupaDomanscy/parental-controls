package env

import (
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
)

func handlePrivateKey(field reflect.Value, envValue string) {
	var decodedHex []byte

	_, err := hex.Decode(decodedHex, []byte(envValue))
	if err != nil {
		panic(err)
	}

	key, err := x509.ParsePKCS1PrivateKey(decodedHex)
	if err != nil {
		panic(err)
	}

	field.Set(reflect.ValueOf(key))
}

func ReadToCfg(cfg interface{}) {
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

		envName := reflect.TypeOf(cfg).Elem().Field(i).Tag.Get("env")
		if envName == "" {
			log.Fatalf("Field %s does not have an env tag.", fieldName)
		}

		envValue := os.Getenv(envName)
		if envValue == "" {
			log.Fatalf("env %s has not been set", envName)
		}

		envTypeTag := reflect.TypeOf(cfg).Elem().Field(i).Tag.Get("env_type")
		if envTypeTag == "RSA_PRIVATE_KEY" {
			handlePrivateKey(field, envValue)
			return
		}

		switch fieldType {
		case reflect.String:
			field.SetString(envValue)
		case reflect.Int:
			intVal, err := strconv.Atoi(envValue)
			if err != nil {
				panic(err)
			}

			field.SetInt(int64(intVal))
		case reflect.Int8:
			intVal, err := strconv.ParseInt(envValue, 10, 8)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Int16:
			intVal, err := strconv.ParseInt(envValue, 10, 16)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Int32:
			intVal, err := strconv.ParseInt(envValue, 10, 32)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Int64:
			intVal, err := strconv.ParseInt(envValue, 10, 64)
			if err != nil {
				panic(err)
			}

			field.SetInt(intVal)
		case reflect.Uint:
			uintVal, err := strconv.ParseUint(envValue, 10, 32)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint8:
			uintVal, err := strconv.ParseUint(envValue, 10, 8)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint16:
			uintVal, err := strconv.ParseUint(envValue, 10, 16)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint32:
			uintVal, err := strconv.ParseUint(envValue, 10, 32)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Uint64:
			uintVal, err := strconv.ParseUint(envValue, 10, 64)
			if err != nil {
				panic(err)
			}

			field.SetUint(uintVal)
		case reflect.Float32:
			floatVal, err := strconv.ParseFloat(envValue, 32)
			if err != nil {
				panic(err)
			}

			field.SetFloat(floatVal)
		case reflect.Float64:
			floatVal, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				panic(err)
			}

			field.SetFloat(floatVal)
		default:
			panic("unhandled default case")
		}
	}
}
