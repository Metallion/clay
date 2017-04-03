package logics

import (
	"github.com/jinzhu/gorm"
	"github.com/qb0C80aE/clay/extension"
	"github.com/qb0C80aE/clay/models"
)

type designLogic struct {
	*BaseLogic
}

func newDesignLogic() *designLogic {
	logic := &designLogic{
		BaseLogic: SharedBaseLogic(),
	}
	return logic
}

func (logic *designLogic) GetSingle(db *gorm.DB, _ string, _ string) (interface{}, error) {

	design := &models.Design{
		Content: map[string]interface{}{},
	}

	designAccessors := extension.GetDesignAccessos()
	for _, accessor := range designAccessors {
		key, value, err := accessor.ExtractFromDesign(db)
		if err != nil {
			return nil, err
		}
		design.Content[key] = value
	}

	return design, nil
}

func (logic *designLogic) Update(db *gorm.DB, _ string, data interface{}) (interface{}, error) {
	design := data.(*models.Design)

	designAccessors := extension.GetDesignAccessos()
	for _, accessor := range designAccessors {
		if err := accessor.DeleteFromDesign(db); err != nil {
			return nil, err
		}
	}
	for _, accessor := range designAccessors {
		if err := accessor.LoadToDesign(db, design); err != nil {
			return nil, err
		}
	}

	return design, nil
}

func (logic *designLogic) Delete(db *gorm.DB, _ string) error {
	designAccessors := extension.GetDesignAccessos()
	for _, accessor := range designAccessors {
		if err := accessor.DeleteFromDesign(db); err != nil {
			return err
		}
	}

	return nil
}

var uniqueDesignLogic = newDesignLogic()

func UniqueDesignLogic() extension.Logic {
	return uniqueDesignLogic
}

func init() {
}
