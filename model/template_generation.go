package model

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	dbpkg "github.com/qb0C80aE/clay/db"
	"github.com/qb0C80aE/clay/extension"
	"github.com/qb0C80aE/clay/helper"
	"github.com/qb0C80aE/clay/logging"
	"github.com/qb0C80aE/clay/util/conversion"
	"github.com/qb0C80aE/clay/util/mapstruct"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	tplpkg "text/template"
)

var parameterRegexp = regexp.MustCompile("p\\[(.+)\\]")

// TemplateGeneration is the model class what represents template generation
type TemplateGeneration struct {
	Base
}

// NewTemplateGeneration creates a template generation model instance
func NewTemplateGeneration() *TemplateGeneration {
	return &TemplateGeneration{}
}

// GetContainerForMigration returns its container for migration, if no need to be migrated, just return null
func (receiver *TemplateGeneration) GetContainerForMigration() (interface{}, error) {
	return nil, nil
}

// GenerateTemplate generates text data based on registered templates
// parameters include either id or name
// actual parameters for template arguments must be included in urlValues as shaped like q[...]=...
func (receiver *TemplateGeneration) GenerateTemplate(db *gorm.DB, parameters gin.Params, urlValues url.Values) (interface{}, error) {
	templateArgumentMap := map[interface{}]*TemplateArgument{}
	templateParameterMap := map[interface{}]interface{}{}

	templateArgumentParameterMap := map[interface{}]interface{}{}
	for key := range urlValues {
		subMatch := parameterRegexp.FindStringSubmatch(key)
		if len(subMatch) == 2 {
			templateArgumentParameterMap[subMatch[1]] = urlValues.Get(key)
		}
	}

	templateModel := NewTemplate()
	templateModelAsContainer := NewTemplate()

	// GenerateTemplate resets db conditions like preloads, so you should use this method in GetSingle or GetMulti only,
	// and note that all conditions go away after this method.
	db = db.New()

	_, idExists := parameters.Get("id")
	templateName, nameExists := parameters.Get("name")

	if idExists {
		newURLValues := url.Values{}
		newURLValues.Set("preloads", "template_arguments")

		dbParameter, err := dbpkg.NewParameter(newURLValues)
		if err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}

		db = dbParameter.SetPreloads(db)

		container, err := templateModel.GetSingle(templateModel, db, parameters, newURLValues, "*")
		if err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}

		if err := mapstruct.RemapToStruct(container, templateModelAsContainer); err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}
	} else if nameExists {
		newURLValues := url.Values{}
		newURLValues.Set("q[name]", templateName)
		newURLValues.Set("preloads", "template_arguments")

		dbParameter, err := dbpkg.NewParameter(newURLValues)
		if err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}

		db = dbParameter.FilterFields(db)
		db = dbParameter.SetPreloads(db)

		container, err := templateModel.GetMulti(templateModel, db, parameters, newURLValues, "*")
		if err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}

		containerValue := reflect.ValueOf(container)
		if containerValue.Len() == 0 {
			logging.Logger().Debug("record not found")
			return nil, errors.New("record not found")
		}

		result := reflect.ValueOf(container).Index(0).Interface()

		if err := mapstruct.RemapToStruct(result, templateModelAsContainer); err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}
	} else {
		logging.Logger().Debug("neither id nor name exists in parameters")
		return nil, errors.New("neither id nor name exists in parameters")
	}

	for _, templateArgument := range templateModelAsContainer.TemplateArguments {
		var err error
		templateArgumentMap[templateArgument.Name] = templateArgument
		switch templateArgument.Type {
		case TemplateArgumentTypeInt:
			templateParameterMap[templateArgument.Name], err = conversion.ToInt64Interface(templateArgument.DefaultValue)
		case TemplateArgumentTypeFloat:
			templateParameterMap[templateArgument.Name], err = conversion.ToFloat64Interface(templateArgument.DefaultValue)
		case TemplateArgumentTypeBool:
			templateParameterMap[templateArgument.Name], err = conversion.ToBooleanInterface(templateArgument.DefaultValue)
		case TemplateArgumentTypeString:
			templateParameterMap[templateArgument.Name] = templateArgument.DefaultValue
		default:
			err = fmt.Errorf("invalid type: %v", templateArgument.Type)
		}

		if err != nil {
			logging.Logger().Debug(err.Error())
			return nil, err
		}
	}

	for key, value := range templateArgumentParameterMap {
		templateArgument, exists := templateArgumentMap[key]
		if !exists {
			logging.Logger().Debugf("the argument related to a parameter %s does not exist", key)
			return nil, fmt.Errorf("the argument related to a parameter %s does not exist", key)
		}

		valueType := reflect.TypeOf(value)
		switch templateArgument.Type {
		case TemplateArgumentTypeInt:
			switch valueType.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				templateParameterMap[key] = int64(value.(int))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				templateParameterMap[key] = int64(value.(uint))
			case reflect.String:
				var err error
				templateParameterMap[key], err = strconv.ParseInt(value.(string), 10, 64)
				if err != nil {
					logging.Logger().Debug(err.Error())
					return nil, fmt.Errorf("parameter type mistmatch, %s should be int, or integer-formatted string, but value is %v", key, value)
				}
			default:
				return nil, fmt.Errorf("parameter type mistmatch, %s should be int, or integer-formatted string, but value is %v", key, value)
			}
		case TemplateArgumentTypeFloat:
			switch valueType.Kind() {
			case reflect.Float32, reflect.Float64:
				templateParameterMap[key] = float64(value.(float64))
			case reflect.String:
				var err error
				templateParameterMap[key], err = strconv.ParseFloat(value.(string), 64)
				if err != nil {
					logging.Logger().Debug(err.Error())
					return nil, fmt.Errorf("parameter type mistmatch, %s should be float, or float-formatted string, but value is %v", key, value)
				}
			default:
				return nil, fmt.Errorf("parameter type mistmatch, %s should be float, or float-formatted string, but value is %v", key, value)
			}
		case TemplateArgumentTypeBool:
			switch valueType.Kind() {
			case reflect.Bool:
				templateParameterMap[key] = value.(bool)
			case reflect.String:
				var err error
				templateParameterMap[key], err = strconv.ParseBool(value.(string))
				if err != nil {
					logging.Logger().Debug(err.Error())
					return nil, fmt.Errorf("parameter type mistmatch, %s should be bool, or bool-formatted string, but value is %v", key, value)
				}
			default:
				return nil, fmt.Errorf("parameter type mistmatch, %s should be bool, or bool-formatted string, but value is %v", key, value)
			}
		case TemplateArgumentTypeString:
			templateParameterMap[key] = fmt.Sprintf("%v", value)
		}
	}

	templateParameterMap["ModelStore"] = db

	tpl := tplpkg.New("template")
	templateFuncMaps := extension.GetRegisteredTemplateFuncMapList()
	for _, templateFuncMap := range templateFuncMaps {
		tpl = tpl.Funcs(templateFuncMap)
	}
	tpl, err := tpl.Parse(templateModelAsContainer.TemplateContent)
	if err != nil {
		logging.Logger().Debug(err.Error())
		return nil, err
	}

	var doc bytes.Buffer
	if err := tpl.Execute(&doc, templateParameterMap); err != nil {
		logging.Logger().Debug(err.Error())
		return nil, err
	}

	result := doc.String()

	return result, nil
}

// GetSingle generates text data based on registered templates
// parameters must be given as p[...]=...
func (receiver *TemplateGeneration) GetSingle(_ extension.Model, db *gorm.DB, parameters gin.Params, urlValues url.Values, _ string) (interface{}, error) {
	return receiver.GenerateTemplate(db, parameters, urlValues)
}

func init() {
	funcMap := tplpkg.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
		"mod": func(a, b int) int { return a % b },
		"int": func(value interface{}) (interface{}, error) {
			return conversion.ToIntInterface(value)
		},
		"int8": func(value interface{}) (interface{}, error) {
			return conversion.ToInt8Interface(value)
		},
		"int16": func(value interface{}) (interface{}, error) {
			return conversion.ToInt16Interface(value)
		},
		"int32": func(value interface{}) (interface{}, error) {
			return conversion.ToInt32Interface(value)
		},
		"int64": func(value interface{}) (interface{}, error) {
			return conversion.ToInt64Interface(value)
		},
		"uint": func(value interface{}) (interface{}, error) {
			return conversion.ToUintInterface(value)
		},
		"uint8": func(value interface{}) (interface{}, error) {
			return conversion.ToUint8Interface(value)
		},
		"uint16": func(value interface{}) (interface{}, error) {
			return conversion.ToUint16Interface(value)
		},
		"uint32": func(value interface{}) (interface{}, error) {
			return conversion.ToUint32Interface(value)
		},
		"uint64": func(value interface{}) (interface{}, error) {
			return conversion.ToUint64Interface(value)
		},
		"float32": func(value interface{}) (interface{}, error) {
			return conversion.ToFloat32Interface(value)
		},
		"float64": func(value interface{}) (interface{}, error) {
			return conversion.ToFloat64Interface(value)
		},
		"string": func(value interface{}) interface{} {
			return conversion.ToStringInterface(value)
		},
		"boolean": func(value interface{}) (interface{}, error) {
			return conversion.ToBooleanInterface(value)
		},
		"split": func(value interface{}, separator string) interface{} {
			data := fmt.Sprintf("%v", value)
			return strings.Split(data, separator)
		},
		"join": func(slice interface{}, separator string) (interface{}, error) {
			interfaceSlice, err := mapstruct.SliceToInterfaceSlice(slice)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			stringSlice := make([]string, len(interfaceSlice))

			for index, item := range interfaceSlice {
				stringSlice[index] = fmt.Sprintf("%v", item)
			}

			return strings.Join(stringSlice, separator), nil
		},
		"slice": func(items ...interface{}) interface{} {
			slice := []interface{}{}
			return append(slice, items...)
		},
		"subslice": func(sliceInterface interface{}, begin int, end int) (interface{}, error) {
			slice, err := mapstruct.SliceToInterfaceSlice(sliceInterface)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			if begin < 0 {
				if end < 0 {
					return slice[:], nil
				}
				return slice[:end], nil
			}
			if end < 0 {
				return slice[begin:], nil
			}
			return slice[begin:end], nil
		},
		"append": func(sliceInterface interface{}, item ...interface{}) (interface{}, error) {
			slice, err := mapstruct.SliceToInterfaceSlice(sliceInterface)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			return append(slice, item...), nil
		},
		"concatenate": func(sliceInterface1 interface{}, sliceInterface2 interface{}) (interface{}, error) {
			slice1, err := mapstruct.SliceToInterfaceSlice(sliceInterface1)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			slice2, err := mapstruct.SliceToInterfaceSlice(sliceInterface2)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			return append(slice1, slice2...), nil
		},
		"fieldslice": func(slice interface{}, fieldName string) ([]interface{}, error) {
			return mapstruct.StructSliceToFieldValueInterfaceSlice(slice, fieldName)
		},
		"sort": func(slice interface{}, order string) ([]interface{}, error) {
			return mapstruct.SortSlice(slice, order)
		},
		"map": func(pairs ...interface{}) (map[interface{}]interface{}, error) {
			if len(pairs)%2 == 1 {
				logging.Logger().Debug("numebr of arguments must be even")
				return nil, fmt.Errorf("numebr of arguments must be even")
			}
			m := make(map[interface{}]interface{}, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				m[pairs[i]] = pairs[i+1]
			}
			return m, nil
		},
		"exists": func(target map[interface{}]interface{}, key interface{}) bool {
			_, exists := target[key]
			return exists
		},
		"put": func(target map[interface{}]interface{}, key interface{}, value interface{}) map[interface{}]interface{} {
			target[key] = value
			return target
		},
		"get": func(target map[interface{}]interface{}, key interface{}) interface{} {
			return target[key]
		},
		"delete": func(target map[interface{}]interface{}, key interface{}) map[interface{}]interface{} {
			delete(target, key)
			return target
		},
		"merge": func(source, destination map[interface{}]interface{}) map[interface{}]interface{} {
			for key, value := range source {
				destination[key] = value
			}
			return destination
		},
		"keys": func(target map[interface{}]interface{}) ([]interface{}, error) {
			return mapstruct.MapToKeySlice(target)
		},
		"hash": func(slice interface{}, keyField string) (map[interface{}]interface{}, error) {
			return mapstruct.StructSliceToInterfaceMap(slice, keyField)
		},
		"slicemap": func(slice interface{}, keyField string) (map[interface{}]interface{}, error) {
			sliceMap, err := mapstruct.StructSliceToInterfaceSliceMap(slice, keyField)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			result := make(map[interface{}]interface{}, len(sliceMap))
			for key, value := range sliceMap {
				result[key] = value
			}
			return result, nil
		},
		"sequence": func(begin, end int) interface{} {
			count := end - begin + 1
			result := make([]int, count)
			for i, j := 0, begin; i < count; i, j = i+1, j+1 {
				result[i] = j
			}
			return result
		},
		"single": func(dbObject interface{}, pathInterface interface{}, queryInterface interface{}) (interface{}, error) {
			path := pathInterface.(string)
			controller, err := extension.GetAssociatedControllerWithPath(path)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			pathElements := strings.Split(strings.Trim(path, "/"), "/")
			resourceName := pathElements[0]
			singleURL, err := controller.GetResourceSingleURL()
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			routeElements := strings.Split(strings.Trim(singleURL, "/"), "/")

			parameters := gin.Params{}
			for index, routeElement := range routeElements {
				if routeElement[:1] == ":" {
					parameter := gin.Param{
						Key:   routeElement[1:],
						Value: pathElements[index],
					}
					parameters = append(parameters, parameter)
				}
			}

			query := queryInterface.(string)
			URL := "/"
			if query != "" {
				URL = "/?" + query
			}
			requestForParameter, err := http.NewRequest(
				http.MethodGet,
				URL,
				nil,
			)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			model, err := extension.GetAssociatedModelWithResourceName(resourceName)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			urlQuery := requestForParameter.URL.Query()
			parameter, err := dbpkg.NewParameter(urlQuery)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			// single resets db conditions like preloads, so you should use this method in GetSingle or GetMulti only,
			// and note that all conditions go away after this method.
			db := dbObject.(*gorm.DB).New()
			db, err = parameter.Paginate(db)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			db = parameter.SetPreloads(db)
			db = parameter.FilterFields(db)
			fields := helper.ParseFields(parameter.DefaultQuery(urlQuery, "fields", "*"))
			queryFields := helper.QueryFields(model, fields)

			result, err := model.GetSingle(model, db, parameters, requestForParameter.URL.Query(), queryFields)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			return result, nil
		},
		"multi": func(dbObject interface{}, pathInterface interface{}, queryInterface interface{}) (interface{}, error) {
			path := pathInterface.(string)
			controller, err := extension.GetAssociatedControllerWithPath(path)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			pathElements := strings.Split(strings.Trim(path, "/"), "/")
			resourceName := pathElements[0]
			multiURL, err := controller.GetResourceMultiURL()
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			routeElements := strings.Split(strings.Trim(multiURL, "/"), "/")

			parameters := gin.Params{}
			for index, routeElement := range routeElements {
				if routeElement[:1] == ":" {
					parameter := gin.Param{
						Key:   routeElement[1:],
						Value: pathElements[index],
					}
					parameters = append(parameters, parameter)
				}
			}

			query := queryInterface.(string)
			URL := "/"
			if query != "" {
				URL = "/?" + query
			}
			requestForParameter, err := http.NewRequest(
				http.MethodGet,
				URL,
				nil,
			)
			model, err := extension.GetAssociatedModelWithResourceName(resourceName)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			urlQuery := requestForParameter.URL.Query()
			parameter, err := dbpkg.NewParameter(urlQuery)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			// multi resets db conditions like preloads, so you should use this method in GetSingle or GetMulti only,
			// and note that all conditions go away after this method.
			db := dbObject.(*gorm.DB).New()
			db, err = parameter.Paginate(db)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			db = parameter.SetPreloads(db)
			db = parameter.SortRecords(db)
			db = parameter.FilterFields(db)
			fields := helper.ParseFields(parameter.DefaultQuery(urlQuery, "fields", "*"))
			queryFields := helper.QueryFields(model, fields)
			result, err := model.GetMulti(model, db, parameters, requestForParameter.URL.Query(), queryFields)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			return result, nil
		},
		"first": func(dbObject interface{}, pathInterface interface{}, queryInterface interface{}) (interface{}, error) {
			path := pathInterface.(string)
			controller, err := extension.GetAssociatedControllerWithPath(path)
			if err != nil {
				return nil, err
			}

			pathElements := strings.Split(strings.Trim(path, "/"), "/")
			resourceName := pathElements[0]
			multiURL, err := controller.GetResourceMultiURL()
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			routeElements := strings.Split(strings.Trim(multiURL, "/"), "/")

			parameters := gin.Params{}
			for index, routeElement := range routeElements {
				if routeElement[:1] == ":" {
					parameter := gin.Param{
						Key:   routeElement[1:],
						Value: pathElements[index],
					}
					parameters = append(parameters, parameter)
				}
			}

			query := queryInterface.(string)
			URL := "/"
			if query != "" {
				URL = "/?" + query
			}
			requestForParameter, err := http.NewRequest(
				http.MethodGet,
				URL,
				nil,
			)
			model, err := extension.GetAssociatedModelWithResourceName(resourceName)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			urlQuery := requestForParameter.URL.Query()
			parameter, err := dbpkg.NewParameter(urlQuery)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			// first resets db conditions like preloads, so you should use this method in GetSingle or GetMulti only,
			// and note that all conditions go away after this method.
			db := dbObject.(*gorm.DB).New()
			db, err = parameter.Paginate(db)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			db = parameter.SetPreloads(db)
			db = parameter.SortRecords(db)
			db = parameter.FilterFields(db)
			fields := helper.ParseFields(parameter.DefaultQuery(urlQuery, "fields", "*"))
			queryFields := helper.QueryFields(model, fields)
			result, err := model.GetMulti(model, db, parameters, requestForParameter.URL.Query(), queryFields)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			resultValue := reflect.ValueOf(result)
			if resultValue.Len() == 0 {
				logging.Logger().Debug("no record selected")
				return nil, errors.New("no record selected")
			}

			return resultValue.Index(0).Interface(), nil
		},
		"total": func(dbObject interface{}, pathInterface interface{}) (interface{}, error) {
			path := pathInterface.(string)
			pathElements := strings.Split(strings.Trim(path, "/"), "/")
			resourceName := pathElements[0]

			model, err := extension.GetAssociatedModelWithResourceName(resourceName)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}

			// total resets db conditions like preloads, so you should use this method in GetSingle or GetMulti only,
			// and note that all conditions go away after this method.
			db := dbObject.(*gorm.DB).New()
			total, err := model.GetTotal(model, db)
			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			return total, nil
		},
		"include": func(dbObject interface{}, templateName string, query string) (interface{}, error) {
			// include resets db conditions like preloads, so you should use this method in GetSingle or GetMulti only,
			// and note that all conditions go away after this method.
			db := dbObject.(*gorm.DB).New()

			parameters := gin.Params{
				{
					Key:   "name",
					Value: templateName,
				},
			}

			urlValues, err := url.ParseQuery(query)

			result, err := NewTemplateGeneration().GetSingle(nil, db, parameters, urlValues, "")

			if err != nil {
				logging.Logger().Debug(err.Error())
				return nil, err
			}
			return result, nil
		},
	}
	extension.RegisterTemplateFuncMap(funcMap)
}

func init() {
	extension.RegisterModel(NewTemplateGeneration())
}
