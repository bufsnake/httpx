package models

type Images struct {
	Id    int    `json:"id" gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY"`
	Image string `json:"image" gorm:"column:image;type:LONGTEXT"`
}
