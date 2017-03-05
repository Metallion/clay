package submodules

import (
	"github.com/qb0C80aE/clay/controllers"
	"github.com/qb0C80aE/clay/logics"
	"github.com/qb0C80aE/clay/models"
	"github.com/qb0C80aE/pottery"
	"github.com/qb0C80aE/loam"
)

func HookSubmodules() {
	controllers.HookSubmodules()
	logics.HookSubmodules()
	models.HookSubmodules()
	pottery.HookSubmodules()
	loam.HookSubmodules()
}
