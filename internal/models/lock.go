package models

type Lock struct {
	Id   int `json:"id" gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY"`
	Lock bool
}

func (Lock) TableName() string {
	return "lock"
}
