package configor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	// "log"
	"os"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func (configor *Configor) getENVPrefix(config interface{}) (prefix string) {
	prefix = configor.Config.ENVPrefix
	if prefix == "" {
		prefix = os.Getenv("CONFIGOR_ENV_PREFIX")

		if prefix == "" {
			prefix = "Configor"
		}
	}
	return
}

func processFile(config interface{}, file string) error {
	data, err := ioutil.ReadFile(file)

	if err != nil {
		return err
	}

	if strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml") {
		return yaml.Unmarshal(data, config)
	}

	if json.Unmarshal(data, config) != nil {
		if yaml.Unmarshal(data, config) != nil {
			return errors.New("failed to decode config")
		}
	}
	return nil
}

func getPrefixForStruct(prefixes []string, fieldStruct *reflect.StructField) []string {
	if fieldStruct.Anonymous && fieldStruct.Tag.Get("anonymous") == "true" {
		return prefixes
	}
	return append(prefixes, fieldStruct.Name)
}

func setEnvNames(envName string, prefixes []string, fieldName string) (envNames []string) {
	if envName == "" {
		envNames = append(envNames, strings.Join(append(prefixes, fieldName), "_"))                  // Configor_DB_Name
		envNames = append(envNames, strings.ToUpper(strings.Join(append(prefixes, fieldName), "_"))) // CONFIGOR_DB_NAME
	} else {
		envNames = []string{envName}
	}
	return
}

func yamlUnmarshalValue(value string, field interface{}) (err error) {
	err = yaml.Unmarshal([]byte(value), field)

	if err != nil {
		return
	}
	return
}

func useDefaultFieldValue(fieldStruct *reflect.StructField, field interface{}) (err error) {
	defaultValue := fieldStruct.Tag.Get("default")

	if defaultValue != "" {
		err = yamlUnmarshalValue(defaultValue, field)
	} else if fieldStruct.Tag.Get("required") == "true" {
		err = errors.New(fieldStruct.Name + " is required, but blank")
	}
	return
}

func processTags(config interface{}, prefixes ...string) error {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()

	for i := 0; i < configType.NumField(); i++ {
		var (
			envNames    []string
			fieldStruct = configType.Field(i)
			field       = configValue.Field(i)
			envName     = fieldStruct.Tag.Get("env") // read configuration from shell env
		)
		envNames = setEnvNames(envName, prefixes, fieldStruct.Name)
		// Load From Shell ENV
		for _, envVarKey := range envNames {
			envVarValue := os.Getenv(envVarKey)

			if envVarValue != "" {
				yamlUnmarshalValue(envVarValue, field.Addr().Interface())
				break
			}
		}
		fieldValueIsBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface())

		if fieldValueIsBlank {
			err := useDefaultFieldValue(&fieldStruct, field.Addr().Interface())

			if err != nil {
				return err
			}
		}
		if field.Kind() == reflect.Struct {
			err := processTags(field.Addr().Interface(), getPrefixForStruct(prefixes, &fieldStruct)...)

			if err != nil {
				return err
			}
		}
		if field.Kind() == reflect.Slice {
			for i := 0; i < field.Len(); i++ {
				if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
					err := processTags(field.Index(i).Addr().Interface(), append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(i))...)

					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
