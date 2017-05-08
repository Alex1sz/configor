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

func (configor *Configor) getENVPrefix(config interface{}) string {
	if configor.Config.ENVPrefix == "" {
		if prefix := os.Getenv("CONFIGOR_ENV_PREFIX"); prefix != "" {
			return prefix
		}
		return "Configor"
	}
	return configor.Config.ENVPrefix
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

func processTags(config interface{}, prefixes ...string) error {
	configValue := reflect.Indirect(reflect.ValueOf(config))

	if configValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	configType := configValue.Type()
	for i := 0; i < configType.NumField(); i++ {
		var (
			envNames    []string
			fieldStruct = configType.Field(i)
			field       = configValue.Field(i)
			envName     = fieldStruct.Tag.Get("env") // read configuration from shell env
		)

		if envName == "" {
			envNames = append(envNames, strings.Join(append(prefixes, fieldStruct.Name), "_"))                  // Configor_DB_Name
			envNames = append(envNames, strings.ToUpper(strings.Join(append(prefixes, fieldStruct.Name), "_"))) // CONFIGOR_DB_NAME
		} else {
			envNames = []string{envName}
		}

		// Load From Shell ENV
		for _, envVarKey := range envNames {
			value := os.Getenv(envVarKey)
			// log.Println("Value...:")
			// log.Println(envName)
			if value != "" {
				err := yaml.Unmarshal([]byte(value), field.Addr().Interface())

				if err != nil {
					return err
				}
				break
			}
		}
		isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface())

		if isBlank {
			// Set default configuration if blank
			value := fieldStruct.Tag.Get("default")

			if value != "" {
				err := yaml.Unmarshal([]byte(value), field.Addr().Interface())

				if err != nil {
					return err
				}
			} else if fieldStruct.Tag.Get("required") == "true" {
				// return error if it is required but blank
				return errors.New(fieldStruct.Name + " is required, but blank")
			}
		}
		// notice weird for conditional below
		// TODO: confirm is pointless and remove
		// for field.Kind() == reflect.Ptr {
		// 	field = field.Elem()
		// }
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
