package models

type Datas struct {
	Id         int    `json:"id" gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY"`
	IP         string `json:"ip" gorm:"column:ip"`
	URL        string `json:"url" gorm:"column:host"`
	Title      string `json:"title" gorm:"column:title"`
	StatusCode string `json:"statuscode" gorm:"column:statuscode"`
	BodyLength string `json:"bodylength" gorm:"column:bodylength"`
	CreateTime string `json:"createtime" gorm:"column:createtime"`
	Image      string `json:"image" gorm:"column:image;type:LONGTEXT"`
	HTTPDump   string `json:"httpdump" gorm:"column:body;type:LONGTEXT"`
	TLS        string `json:"tls" gorm:"column:tls;type:LONGTEXT"`
	ICP        string `json:"icp" gorm:"column:icp"`
}

func (Datas) TableName() string {
	return "datas"
}
