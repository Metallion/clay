package models

type NodePv struct {
	ID   int    `json:"id" gorm:"primary_key"`
	Name string `json:"name" gorm:"not null"`
}

var NodePvModel = &NodePv{}