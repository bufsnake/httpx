package models

type Datas struct {
	Id         int      `json:"id" gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY"`
	IP         string   `json:"ip" gorm:"column:ip"`
	URL        string   `json:"url" gorm:"column:host"`
	Title      string   `json:"title" gorm:"column:title"`
	StatusCode string   `json:"statuscode" gorm:"column:statuscode"`
	BodyLength string   `json:"bodylength" gorm:"column:bodylength"`
	CreateTime string   `json:"createtime" gorm:"column:createtime"`
	Image      string   `json:"image" gorm:"column:image;type:LONGTEXT"`
	HTTPDump   string   `json:"httpdump" gorm:"column:body;type:LONGTEXT"`
	TLS        string   `json:"tls" gorm:"column:tls;type:LONGTEXT"`
	ICP        string   `json:"icp" gorm:"column:icp"`
	Fingers    []Finger `json:"fingers" gorm:"-"`
}

func (Datas) TableName() string {
	return "datas"
}

type Finger struct {
	Id         int    `json:"id" gorm:"column:id;AUTO_INCREMENT;NOT NULL;PRIMARY KEY"`
	URL        string `json:"url" gorm:"column:url"`
	Name       string `json:"name" gorm:"column:name"`
	Confidence int    `json:"confidence" gorm:"column:confidence"`
	Version    string `json:"version" gorm:"column:version"`
	ICON       string `json:"icon" gorm:"column:icon;type:LONGTEXT"`
	WebSite    string `json:"website" gorm:"column:website"`
	CPE        string `json:"cpe" gorm:"column:cpe"`
	// a+\n+b+\n+c
	Categories string `json:"categories" gorm:"column:categories;type:LONGTEXT"`
}

func (Finger) TableName() string {
	return "finger"
}
