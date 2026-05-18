package service

import (
	"context"
	"strings"
	"time"
)

const (
	newLawExpressLimit  = 6
	newLawWithinMonths  = 6
	dateLayoutForCutoff = "2006-01-02"
)

type HomeLawsResponse struct {
	NewLawExpress NewLawExpressSection `json:"newLawExpress"`
	LawCategories LawCategoriesSection `json:"lawCategories"`
	CommonLaws    CommonLawsSection    `json:"commonLaws"`
}

type NewLawExpressSection struct {
	SectionTitle string       `json:"sectionTitle"`
	Items        []NewLawItem `json:"items"`
}

type NewLawItem struct {
	UUID           string `json:"uuid"`
	LawType        string `json:"law_type"`
	LawTypeDisplay string `json:"law_type_display"`
	Subtitle       string `json:"subtitle"`
	BgColor        string `json:"bgColor"`
	LinkUrl        string `json:"linkUrl"`
	PublishDate    string `json:"publishDate"`    // 发布日期 (年月日)
	EffectDate     string `json:"effectDate"`     // 生效日期
	AuthorityName  string `json:"authorityName"`  // 发布机关
}

type LawCategoriesSection struct {
	SectionTitle string            `json:"sectionTitle"`
	Items        []LawCategoryItem `json:"items"`
}

type LawCategoryItem struct {
	UUID           string `json:"uuid"`
	TypeID         int    `json:"type_id"`
	LawType        string `json:"law_type"`
	LawTypeDisplay string `json:"law_type_display"`
	Icon           string `json:"icon"`
}

type CommonLawsSection struct {
	SectionTitle string          `json:"sectionTitle"`
	Items        []CommonLawItem `json:"items"`
}

type CommonLawItem struct {
	UUID           string `json:"uuid"`
	CommonTypeID   int    `json:"common_type_id"` // 对应 common_law_type 表的 id
	TypeID         int    `json:"type_id"`
	LawType        string `json:"law_type"`
	LawTypeDisplay string `json:"law_type_display"`
	Icon           string `json:"icon"`
	Count          int    `json:"count"`
}

var newLawBgColors = []string{"#BF4A90D9", "#BF5C7A9E", "#BF7A5C9E"}

// 与移动端 mock 数据 (laws_page.json) 保持一致：law_type / icon / uuid 是固定映射，
// 仅 law_type_display 走数据库 types 表，方便后续命名调整自动生效。
type lawCategoryDef struct {
	TypeID  int
	UUID    string
	LawType string
	Icon    string
}

var lawCategoryDefs = []lawCategoryDef{
	{TypeID: 100, UUID: "22222222-2222-2222-2222-000000000001", LawType: "constitution", Icon: "account_balance"},
	{TypeID: 110, UUID: "22222222-2222-2222-2222-000000000002", LawType: "constitution_related", Icon: "menu_book"},
	{TypeID: 160, UUID: "22222222-2222-2222-2222-000000000003", LawType: "criminal_law", Icon: "security"},
	{TypeID: 120, UUID: "22222222-2222-2222-2222-000000000004", LawType: "civil_commercial_law", Icon: "groups"},
	{TypeID: 170, UUID: "22222222-2222-2222-2222-000000000005", LawType: "procedural_law", Icon: "gavel"},
	{TypeID: 130, UUID: "22222222-2222-2222-2222-000000000006", LawType: "administrative_law", Icon: "business_center"},
	{TypeID: 140, UUID: "22222222-2222-2222-2222-000000000007", LawType: "economic_law", Icon: "account_balance_wallet"},
	{TypeID: 150, UUID: "22222222-2222-2222-2222-000000000008", LawType: "social_law", Icon: "favorite"},
}

// commonLawDef 是「常用法律」section 的静态定义；Count 在请求时按 Keywords 在 laws_list.title 上
// 用 `LIKE OR` 聚合计数得到。Keywords 是粗粒度匹配，可能少量误命中或重复命中（多分类之间允许重叠）。
type commonLawDef struct {
	UUID           string
	TypeID         int
	LawType        string
	LawTypeDisplay string
	Icon           string
	Keywords       []string
}

var commonLawDefs = []commonLawDef{
	{
		UUID: "33333333-3333-3333-3333-000000000001", TypeID: 120,
		LawType: "marriage_family", LawTypeDisplay: "婚姻家庭", Icon: "family_restroom",
		Keywords: []string{"婚姻", "家庭", "收养", "继承", "监护", "结婚", "离婚"},
	},
	{
		UUID: "33333333-3333-3333-3333-000000000002", TypeID: 120,
		LawType: "goods_trading", LawTypeDisplay: "商品买卖", Icon: "shopping_cart",
		Keywords: []string{"消费者", "产品质量", "食品安全", "药品", "买卖", "商品"},
	},
	{
		UUID: "33333333-3333-3333-3333-000000000003", TypeID: 150,
		LawType: "labor_personnel", LawTypeDisplay: "劳动人事", Icon: "work",
		Keywords: []string{"劳动", "就业", "工会", "工伤", "职工", "社会保险"},
	},
	{
		UUID: "33333333-3333-3333-3333-000000000004", TypeID: 130,
		LawType: "traffic_regulations", LawTypeDisplay: "交通法规", Icon: "directions_car",
		Keywords: []string{"交通", "公路", "铁路", "航空", "航运", "港口", "机动车"},
	},
	{
		UUID: "33333333-3333-3333-3333-000000000005", TypeID: 120,
		LawType: "loan_guarantee", LawTypeDisplay: "借贷担保", Icon: "credit_score",
		Keywords: []string{"银行", "贷款", "担保", "借贷", "信贷"},
	},
	{
		UUID: "33333333-3333-3333-3333-000000000006", TypeID: 130,
		LawType: "public_security_case", LawTypeDisplay: "治安案件", Icon: "local_police",
		Keywords: []string{"治安", "公安", "消防", "集会游行", "出入境"},
	},
	{
		UUID: "33333333-3333-3333-3333-000000000007", TypeID: 160,
		LawType: "criminal_case", LawTypeDisplay: "刑事案件", Icon: "gavel",
		Keywords: []string{"刑法", "刑事", "刑罚", "反恐怖", "反间谍", "禁毒", "反洗钱"},
	},
}

func (s *LawService) GetHomeLaws(ctx context.Context) (*HomeLawsResponse, error) {
	now := time.Now()
	return s.buildHomeLaws(ctx, now)
}

func (s *LawService) buildHomeLaws(ctx context.Context, now time.Time) (*HomeLawsResponse, error) {
	today := now.Format(dateLayoutForCutoff)
	cutoff := now.AddDate(0, -newLawWithinMonths, 0).Format(dateLayoutForCutoff)

	newLaws, err := s.lawRepo.ListNewLaws(ctx, cutoff, today, newLawExpressLimit)
	if err != nil {
		return nil, err
	}

	newLawItems := make([]NewLawItem, 0, len(newLaws))
	for i, law := range newLaws {
		newLawItems = append(newLawItems, NewLawItem{
			UUID:           law.VersionID,
			LawType:        law.LawType,
			LawTypeDisplay: law.Title,
			Subtitle:       buildNewLawSubtitle(law.PublishDate, law.AuthorityName),
			BgColor:        newLawBgColors[i%len(newLawBgColors)],
			LinkUrl:        "",
			PublishDate:    law.PublishDate,
			EffectDate:     law.EffectDate,
			AuthorityName:  law.AuthorityName,
		})
	}

	categoryIDs := make([]int, 0, len(lawCategoryDefs))
	for _, def := range lawCategoryDefs {
		categoryIDs = append(categoryIDs, def.TypeID)
	}
	types, err := s.typeRepo.ListByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	typeNameByID := make(map[int]string, len(types))
	for _, t := range types {
		typeNameByID[t.ID] = t.Name
	}

	categoryItems := make([]LawCategoryItem, 0, len(lawCategoryDefs))
	for _, def := range lawCategoryDefs {
		name, ok := typeNameByID[def.TypeID]
		if !ok {
			continue
		}
		categoryItems = append(categoryItems, LawCategoryItem{
			UUID:           def.UUID,
			TypeID:         def.TypeID,
			LawType:        def.LawType,
			LawTypeDisplay: name,
			Icon:           def.Icon,
		})
	}

	// 从 common_law_type 和 common_laws 表查询常用法律数据
	commonTypes, err := s.commonLawRepo.ListAllTypes(ctx)
	if err != nil {
		return nil, err
	}

	counts, err := s.commonLawRepo.GetTypesCount(ctx)
	if err != nil {
		return nil, err
	}

	commonItems := make([]CommonLawItem, 0, len(commonTypes))
	for _, t := range commonTypes {
		commonItems = append(commonItems, CommonLawItem{
			UUID:           t.UUID,
			CommonTypeID:   t.ID,
			TypeID:         t.TypeID,
			LawType:        t.LawType,
			LawTypeDisplay: t.LawTypeDisplay,
			Icon:           t.Icon,
			Count:          int(counts[t.ID]),
		})
	}

	return &HomeLawsResponse{
		NewLawExpress: NewLawExpressSection{
			SectionTitle: "新法速递",
			Items:        newLawItems,
		},
		LawCategories: LawCategoriesSection{
			SectionTitle: "法律概览",
			Items:        categoryItems,
		},
		CommonLaws: CommonLawsSection{
			SectionTitle: "常用法律",
			Items:        commonItems,
		},
	}, nil
}

func buildNewLawSubtitle(publishDate, authorityName string) string {
	publishDate = strings.TrimSpace(publishDate)
	if len(publishDate) >= 4 {
		return publishDate[:4] + "年发布"
	}

	authorityName = strings.TrimSpace(authorityName)
	if authorityName != "" {
		return authorityName + "发布"
	}

	return ""
}
