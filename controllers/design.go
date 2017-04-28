package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qb0C80aE/clay/db"
	"github.com/qb0C80aE/clay/extensions"
	"github.com/qb0C80aE/clay/logics"
	"github.com/qb0C80aE/clay/models"
)

type designController struct {
	*BaseController
}

func newDesignController() extensions.Controller {
	controller := &designController{
		BaseController: NewBaseController(
			models.SharedDesignModel(),
			logics.UniqueDesignLogic(),
		),
	}
	controller.SetOutputter(controller)
	return controller
}

func (controller *designController) RouteMap() map[int]map[string]gin.HandlerFunc {
	url := fmt.Sprintf("%s/present", controller.ResourceName())
	routeMap := map[int]map[string]gin.HandlerFunc{
		extensions.MethodGet: {
			url: controller.GetSingle,
		},
		extensions.MethodPut: {
			url: controller.Update,
		},
		extensions.MethodDelete: {
			url: controller.Delete,
		},
	}
	return routeMap
}

func (controller *designController) Update(c *gin.Context) {
	db.Instance(c).Exec("pragma foreign_keys = off;")
	controller.BaseController.Update(c)
	db.Instance(c).Exec("pragma foreign_keys = on;")
}

func (controller *designController) Delete(c *gin.Context) {
	db.Instance(c).Exec("pragma foreign_keys = off;")
	controller.BaseController.Delete(c)
	db.Instance(c).Exec("pragma foreign_keys = on;")
}

var uniqueDesignController = newDesignController()

func init() {
	extensions.RegisterController(uniqueDesignController)
}
