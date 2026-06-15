package model

// LocalAuthority 地方法律制定机关，按省/市两级归属存储，供客户端三级联动选择
type LocalAuthority struct {
	ID            int    `gorm:"column:id;primaryKey;autoIncrement"`
	AuthorityName string `gorm:"column:authorityName;uniqueIndex;not null"`
	Province      string `gorm:"column:province"` // 省级行政区，如"广东省""上海市""内蒙古自治区"
	City          string `gorm:"column:city"`      // 市/州名，如"深圳市""凉山彝族自治州"；省级机关为空
	LawCount      int64  `gorm:"column:lawCount"`  // 该机关下地方法律条数
}

func (LocalAuthority) TableName() string {
	return "local_authority_list"
}
