package controllers

import (
	dbpkg "github.com/qb0C80aE/clay/db"
	"github.com/qb0C80aE/clay/helper"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/qb0C80aE/clay/extensions"
	"io/ioutil"
	"net/http"
	"reflect"
)

// BaseController is the base class that all controller classes inherit
type BaseController struct {
	resourceName string
	model        interface{}
	logic        extensions.Logic
	outputter    extensions.Outputter
}

// NewBaseController creates a new instance of BaseController
func NewBaseController(resourceName string, model interface{}, logic extensions.Logic) *BaseController {
	controller := &BaseController{
		resourceName: resourceName,
		model:        model,
		logic:        logic,
	}
	return controller
}

func (controller *BaseController) bind(c *gin.Context, container interface{}) error {
	if err := c.Bind(container); err != nil {
		return err
	}
	if c.Request.ParseMultipartForm(1024*1024) == nil {
		files := c.Request.MultipartForm.File
		for name := range files {
			buffer := &bytes.Buffer{}
			file, _, err := c.Request.FormFile(name)
			if err != nil {
				return err
			}

			data, err := ioutil.ReadAll(file)
			if err != nil {
				return err
			}

			_, err = buffer.Write(data)
			if err != nil {
				return err
			}

			vs := reflect.ValueOf(container)
			for vs.Kind() == reflect.Ptr {
				vs = vs.Elem()
			}
			if !vs.IsValid() {
				return errors.New("invalid container")
			}
			if !vs.CanInterface() {
				return errors.New("invalid container")
			}
			value := vs.Interface()

			t := reflect.TypeOf(value)
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				jsonTag := field.Tag.Get("json")
				formTag := field.Tag.Get("form")
				if (jsonTag == name || formTag == name) && (field.Type.Kind() == reflect.String) {
					vs.FieldByName(field.Name).SetString(buffer.String())
					break
				}
			}

		}
	}
	return nil
}

// ResourceName returns its resource name in REST
func (controller *BaseController) ResourceName() string {
	return controller.resourceName
}

// OutputError handles an error output
func (controller *BaseController) OutputError(c *gin.Context, code int, err error) {
	c.JSON(code, gin.H{"error": err.Error()})
}

// OutputGetSingle corresponds HTTP GET message and handles the output of a single result from logic classes
func (controller *BaseController) OutputGetSingle(c *gin.Context, code int, result interface{}, fields map[string]interface{}) {
	if fields == nil {
		c.JSON(code, result)
	} else {
		fieldMap, err := helper.FieldToMap(result, fields)
		if err != nil {
			controller.OutputError(c, http.StatusBadRequest, err)
			return
		}

		if _, ok := c.GetQuery("pretty"); ok {
			c.IndentedJSON(code, fieldMap)
		} else {
			c.JSON(code, fieldMap)
		}
	}
}

// OutputGetMulti corresponds HTTP GET message and handles the output of multiple result from logic classes
func (controller *BaseController) OutputGetMulti(c *gin.Context, code int, result interface{}, fields map[string]interface{}) {
	if fields == nil {
		c.JSON(code, result)
	} else {
		v := reflect.ValueOf(result)

		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			controller.OutputError(c, http.StatusBadRequest, errors.New("given argument is neither a slice nor an array"))
			return
		}

		size := v.Len()

		if _, ok := c.GetQuery("stream"); ok {
			enc := json.NewEncoder(c.Writer)
			c.Status(code)

			for i := 0; i < size; i++ {
				item := v.Index(i)

				if !item.CanInterface() {
					controller.OutputError(c, http.StatusBadRequest, fmt.Errorf("the original item indexed %d in given slice cannot interface", i))
					return
				}

				fieldMap, err := helper.FieldToMap(item.Interface(), fields)

				if err != nil {
					controller.OutputError(c, http.StatusBadRequest, err)
					return
				}

				if err := enc.Encode(fieldMap); err != nil {
					controller.OutputError(c, http.StatusBadRequest, err)
					return
				}
			}
		} else {
			fieldMaps := []map[string]interface{}{}

			for i := 0; i < size; i++ {
				item := v.Index(i)

				if !item.CanInterface() {
					controller.OutputError(c, http.StatusBadRequest, fmt.Errorf("the original item indexed %d in given slice cannot interface", i))
					return
				}

				fieldMap, err := helper.FieldToMap(item.Interface(), fields)

				if err != nil {
					controller.OutputError(c, http.StatusBadRequest, err)
					return
				}

				fieldMaps = append(fieldMaps, fieldMap)
			}

			if _, ok := c.GetQuery("pretty"); ok {
				c.IndentedJSON(code, fieldMaps)
			} else {
				c.JSON(code, fieldMaps)
			}
		}
	}
}

// OutputCreate corresponds HTTP POST message and handles the output of a single result from logic classes
func (controller *BaseController) OutputCreate(c *gin.Context, code int, result interface{}) {
	controller.OutputGetSingle(c, code, result, nil)
}

// OutputUpdate corresponds HTTP PUT message and handles the output of a single result from logic classes
func (controller *BaseController) OutputUpdate(c *gin.Context, code int, result interface{}) {
	controller.OutputGetSingle(c, code, result, nil)
}

// OutputDelete corresponds HTTP DELETE message and handles the code result from logic classes
func (controller *BaseController) OutputDelete(c *gin.Context, code int) {
	c.Writer.WriteHeader(code)
}

// OutputPatch corresponds HTTP PATCH message and handles the output of a single result from logic classes
func (controller *BaseController) OutputPatch(c *gin.Context, code int, result interface{}) {
	controller.OutputGetSingle(c, code, result, nil)
}

// OutputOptions corresponds HTTP DELETE message and handles the code result from logic classes, as well as OutputDelete
func (controller *BaseController) OutputOptions(c *gin.Context, code int) {
	controller.OutputDelete(c, code)
}

func (controller *BaseController) getSingle(db *gorm.DB, id string, queryFields string) (result interface{}, err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	result, err = controller.logic.GetSingle(db, id, queryFields)
	return result, err
}

func (controller *BaseController) getMulti(db *gorm.DB, queryFields string) (result interface{}, err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	result, err = controller.logic.GetMulti(db, queryFields)
	return result, err
}

func (controller *BaseController) create(db *gorm.DB, data interface{}) (result interface{}, err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	result, err = controller.logic.Create(db, data)
	return result, err
}

func (controller *BaseController) update(db *gorm.DB, id string, data interface{}) (result interface{}, err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	result, err = controller.logic.Update(db, id, data)
	return result, err
}

func (controller *BaseController) delete(db *gorm.DB, id string) (err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	err = controller.logic.Delete(db, id)
	return err
}

func (controller *BaseController) patch(db *gorm.DB, id string) (result interface{}, err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	result, err = controller.logic.Patch(db, id)
	return result, err
}

func (controller *BaseController) options(db *gorm.DB) (err error) {
	defer func() {
		if recoverResult := recover(); recoverResult != nil {
			err = fmt.Errorf("%v", recoverResult)
		}
	}()

	err = controller.logic.Options(db)
	return err
}

// GetSingle corresponds HTTP GET message and handles a request for a single resource to get the information
func (controller *BaseController) GetSingle(c *gin.Context) {
	id := c.Params.ByName("id")
	db := dbpkg.Instance(c)
	db = dbpkg.SetPreloads(c.Query("preloads"), db)
	fields := helper.ParseFields(c.DefaultQuery("fields", "*"))
	queryFields := helper.QueryFields(controller.model, fields)

	result, err := controller.getSingle(db, id, queryFields)
	if err != nil {
		controller.OutputError(c, http.StatusNotFound, err)
		return
	}

	controller.OutputGetSingle(c, http.StatusOK, result, fields)
}

// GetMulti corresponds HTTP GET message and handles a request for multi resource to get the list of information
func (controller *BaseController) GetMulti(c *gin.Context) {
	db := dbpkg.Instance(c)
	db = dbpkg.SetPreloads(c.Query("preloads"), db)
	db = dbpkg.SortRecords(c.Query("sort"), db)
	db = dbpkg.FilterFields(c, controller.model, db)
	fields := helper.ParseFields(c.DefaultQuery("fields", "*"))
	queryFields := helper.QueryFields(controller.model, fields)

	result, err := controller.getMulti(db, queryFields)
	if err != nil {
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	controller.outputter.OutputGetMulti(c, http.StatusOK, result, fields)
}

// Create corresponds HTTP POST message and handles a request for multi resource to create a new information
func (controller *BaseController) Create(c *gin.Context) {
	vs := reflect.ValueOf(controller.model)
	for vs.Kind() == reflect.Ptr {
		vs = vs.Elem()
	}
	if !vs.IsValid() {
		controller.OutputError(c, http.StatusBadRequest, errors.New("Invalid model"))
		return
	}
	if !vs.CanInterface() {
		controller.OutputError(c, http.StatusBadRequest, errors.New("Invalid model"))
		return
	}
	container := reflect.New(reflect.TypeOf(vs.Interface())).Interface()

	if err := controller.bind(c, container); err != nil {
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db := dbpkg.Instance(c)

	db = db.Begin()
	result, err := controller.create(db, container)
	if err != nil {
		db.Rollback()
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db.Commit()

	controller.outputter.OutputCreate(c, http.StatusCreated, result)
}

// Update corresponds HTTP PUT message and handles a request for a single resource to update the specific information
func (controller *BaseController) Update(c *gin.Context) {
	vs := reflect.ValueOf(controller.model)
	for vs.Kind() == reflect.Ptr {
		vs = vs.Elem()
	}
	if !vs.IsValid() {
		controller.OutputError(c, http.StatusBadRequest, errors.New("Invalid model"))
		return
	}
	if !vs.CanInterface() {
		controller.OutputError(c, http.StatusBadRequest, errors.New("Invalid model"))
		return
	}
	container := reflect.New(reflect.TypeOf(vs.Interface())).Interface()

	id := c.Params.ByName("id")

	if err := controller.bind(c, container); err != nil {
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db := dbpkg.Instance(c)

	db = db.Begin()
	result, err := controller.update(db, id, container)
	if err != nil {
		db.Rollback()
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db.Commit()

	controller.outputter.OutputUpdate(c, http.StatusOK, result)
}

// Delete corresponds HTTP DELETE message and handles a request for a single resource to delete the specific information
func (controller *BaseController) Delete(c *gin.Context) {
	id := c.Params.ByName("id")

	db := dbpkg.Instance(c)

	db = db.Begin()
	err := controller.delete(db, id)
	if err != nil {
		db.Rollback()
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db.Commit()

	controller.outputter.OutputDelete(c, http.StatusNoContent)
}

// Patch corresponds HTTP PATCH message and handles a request for a single resource to update partially the specific information
func (controller *BaseController) Patch(c *gin.Context) {
	id := c.Params.ByName("id")

	db := dbpkg.Instance(c)

	db = db.Begin()
	result, err := controller.patch(db, id)
	if err != nil {
		db.Rollback()
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db.Commit()

	controller.outputter.OutputPatch(c, http.StatusOK, result)
}

// Options corresponds HTTP OPTIONS message and handles a request for multi resources to retrieve its supported options
func (controller *BaseController) Options(c *gin.Context) {
	db := dbpkg.Instance(c)

	db = db.Begin()
	err := controller.options(db)
	if err != nil {
		db.Rollback()
		controller.OutputError(c, http.StatusBadRequest, err)
		return
	}

	db.Commit()

	controller.outputter.OutputOptions(c, http.StatusNoContent)
}

// SetOutputter sets an outputter for this controller
func (controller *BaseController) SetOutputter(outputter extensions.Outputter) {
	controller.outputter = outputter
}