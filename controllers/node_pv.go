package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/qb0C80aE/clay/extension"
	"github.com/qb0C80aE/clay/logics"
	"github.com/qb0C80aE/clay/models"
)

type NodePvController struct {
	BaseController
}

func init() {
	extension.RegisterController(NewNodePvController())
}

func NewNodePvController() *NodePvController {
	controller := &NodePvController{}
	controller.Initialize()
	return controller
}

func (this *NodePvController) Initialize() {
	this.ResourceName = "node_pv"
	this.Model = models.NodePvModel
	this.Logic = logics.NewNodePvLogic()
	this.Outputter = this
}

func (this *NodePvController) GetRouteMap() map[int]map[string]gin.HandlerFunc {
	resourceSingleUrl := extension.GetResourceSingleUrl(this.ResourceName)
	resourceMultiUrl := extension.GetResourceMultiUrl(this.ResourceName)

	routeMap := map[int]map[string]gin.HandlerFunc{
		extension.MethodGet: {
			resourceSingleUrl: this.GetSingle,
			resourceMultiUrl:  this.GetMulti,
		},
		extension.MethodPost: {
			resourceMultiUrl: this.Create,
		},
		extension.MethodPut: {
			resourceSingleUrl: this.Update,
		},
		extension.MethodDelete: {
			resourceSingleUrl: this.Delete,
		},
	}
	return routeMap
}
