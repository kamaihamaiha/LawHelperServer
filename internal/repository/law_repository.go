package repository

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"LawHelperServer/internal/model"
)

const lawListOrder = `
CASE WHEN effectDate IS NULL OR TRIM(effectDate) = '' THEN 1 ELSE 0 END ASC,
effectDate DESC,
CASE WHEN publishDate IS NULL OR TRIM(publishDate) = '' THEN 1 ELSE 0 END ASC,
publishDate DESC,
versionId DESC
`

type LawRepository struct {
	db *gorm.DB
}

type LawSearchFilter struct {
	// repository 层只关心已经标准化后的过滤条件
	Scope             string
	Query             string
	TextMode          string
	AuthorityNames    []string
	EffectiveStatuses []int
	PublishDateStart  string
	PublishDateEnd    string
	EffectDateStart   string
	EffectDateEnd     string
	Offset            int
	Limit             int
}

func NewLawRepository(db *gorm.DB) *LawRepository {
	return &LawRepository{db: db}
}

func (r *LawRepository) SearchLaws(ctx context.Context, filter LawSearchFilter) ([]model.LawSummary, error) {
	var laws []model.LawSummary

	// 搜索结果仍然返回 LawSummary，方便直接复用现有列表卡片和详情跳转
	query := r.buildSearchQuery(ctx, filter).
		Select("l.versionId, l.title, l.lawTypeId, l.lawType, l.publishDate, l.effectDate, l.effectiveStatus, l.authorityName").
		Order(searchLawListOrder)
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if err := query.Find(&laws).Error; err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) CountSearchLaws(ctx context.Context, filter LawSearchFilter) (int64, error) {
	var total int64
	if err := r.buildSearchQuery(ctx, filter).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *LawRepository) ListByType(ctx context.Context, typeID, offset, limit int) ([]model.LawSummary, error) {
	var laws []model.LawSummary

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId, lawType, publishDate, effectDate, effectiveStatus, authorityName").
		Where("lawTypeId = ?", typeID).
		Order(lawListOrder).
		Offset(offset).
		Limit(limit).
		Find(&laws).Error
	if err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) ListAllByType(ctx context.Context, typeID int) ([]model.LawSummary, error) {
	var laws []model.LawSummary

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId, lawType, publishDate, effectDate, effectiveStatus, authorityName").
		Where("lawTypeId = ?", typeID).
		Order(lawListOrder).
		Find(&laws).Error
	if err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) ListNewLaws(ctx context.Context, publishCutoff, today string, limit int) ([]model.LawSummary, error) {
	var laws []model.LawSummary

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId, lawType, publishDate, effectDate, effectiveStatus, authorityName").
		Where(
			"(TRIM(COALESCE(publishDate, '')) != '' AND publishDate >= ?) OR (TRIM(COALESCE(effectDate, '')) = '' OR effectDate > ?)",
			publishCutoff, today,
		).
		Order(`
			CASE WHEN publishDate IS NULL OR TRIM(publishDate) = '' THEN 1 ELSE 0 END ASC,
			publishDate DESC,
			versionId DESC
		`).
		Limit(limit).
		Find(&laws).Error
	if err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) ListNewLawsPaginated(ctx context.Context, publishCutoff, today string, offset, limit int) ([]model.LawSummary, error) {
	var laws []model.LawSummary

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId, lawType, publishDate, effectDate, effectiveStatus, authorityName").
		Where(
			"(TRIM(COALESCE(publishDate, '')) != '' AND publishDate >= ?) OR (TRIM(COALESCE(effectDate, '')) = '' OR effectDate > ?)",
			publishCutoff, today,
		).
		Order(`
			CASE WHEN publishDate IS NULL OR TRIM(publishDate) = '' THEN 1 ELSE 0 END ASC,
			publishDate DESC,
			versionId DESC
		`).
		Offset(offset).
		Limit(limit).
		Find(&laws).Error
	if err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) CountNewLaws(ctx context.Context, publishCutoff, today string) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Where(
			"(TRIM(COALESCE(publishDate, '')) != '' AND publishDate >= ?) OR (TRIM(COALESCE(effectDate, '')) = '' OR effectDate > ?)",
			publishCutoff, today,
		).
		Count(&total).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *LawRepository) CountByTitleKeywords(ctx context.Context, keywords []string) (int64, error) {
	if len(keywords) == 0 {
		return 0, nil
	}

	conditions := make([]string, 0, len(keywords))
	args := make([]any, 0, len(keywords))
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		conditions = append(conditions, "title LIKE ?")
		args = append(args, "%"+kw+"%")
	}
	if len(conditions) == 0 {
		return 0, nil
	}

	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Where(strings.Join(conditions, " OR "), args...).
		Count(&total).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *LawRepository) CountByType(ctx context.Context, typeID int) (int64, error) {
	var total int64

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Where("lawTypeId = ?", typeID).
		Count(&total).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *LawRepository) ListByTypeIDs(ctx context.Context, typeIDs []int, offset, limit int) ([]model.LawSummary, error) {
	if len(typeIDs) == 0 {
		return nil, nil
	}

	var laws []model.LawSummary

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId, lawType, publishDate, effectDate, effectiveStatus, authorityName").
		Where("lawTypeId IN ?", typeIDs).
		Order(lawListOrder).
		Offset(offset).
		Limit(limit).
		Find(&laws).Error
	if err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) ListAllByTypeIDs(ctx context.Context, typeIDs []int) ([]model.LawSummary, error) {
	if len(typeIDs) == 0 {
		return nil, nil
	}

	var laws []model.LawSummary

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId, lawType, publishDate, effectDate, effectiveStatus, authorityName").
		Where("lawTypeId IN ?", typeIDs).
		Order(lawListOrder).
		Find(&laws).Error
	if err != nil {
		return nil, err
	}

	return laws, nil
}

func (r *LawRepository) CountByTypeIDs(ctx context.Context, typeIDs []int) (int64, error) {
	if len(typeIDs) == 0 {
		return 0, nil
	}

	var total int64

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Where("lawTypeId IN ?", typeIDs).
		Count(&total).Error
	if err != nil {
		return 0, err
	}

	return total, nil
}

type bigGroupRow struct {
	BigGroup      string `gorm:"column:big_group"`
	TypeID        int    `gorm:"column:type_id"`
	TypeName      string `gorm:"column:type_name"`
	Count         int64  `gorm:"column:cnt"`
	Rank          int    `gorm:"column:rn"`
	TotalSub      int    `gorm:"column:total_sub"`
	BigGroupTotal int64  `gorm:"column:big_group_total"`
	SortKey       int    `gorm:"column:sort_key"`
}

func (r *LawRepository) ListBigGroupStats(ctx context.Context) ([]model.BigGroupStat, error) {
	const query = `
WITH sub_counts AS (
    SELECT
        CASE COALESCE(t_parent.id, t.id)
            WHEN 100 THEN '宪法'
            WHEN 101 THEN '法律'
            WHEN 102 THEN '法律'
            WHEN 210 THEN '行政法规'
            WHEN 220 THEN '监察法规'
            WHEN 222 THEN '地方法规'
            WHEN 320 THEN '司法解释'
            WHEN 330 THEN '司法解释'
            WHEN 340 THEN '司法解释'
            WHEN 350 THEN '司法解释'
        END AS big_group,
        MIN(COALESCE(t_parent.id, t.id)) AS sort_key,
        t.id  AS type_id,
        t.name AS type_name,
        COUNT(*) AS cnt
    FROM laws_list l
    JOIN types t ON l.lawTypeId = t.id
    LEFT JOIN types t_parent ON t.parent_id = t_parent.id
    GROUP BY big_group, t.id, t.name
),
ranked AS (
    SELECT
        big_group,
        sort_key,
        type_id,
        type_name,
        cnt,
        ROW_NUMBER() OVER (PARTITION BY big_group ORDER BY cnt DESC) AS rn,
        COUNT(*)     OVER (PARTITION BY big_group)                   AS total_sub,
        SUM(cnt)     OVER (PARTITION BY big_group)                   AS big_group_total
    FROM sub_counts
)
SELECT big_group, sort_key, type_id, type_name, cnt, rn, total_sub, big_group_total
FROM ranked
ORDER BY sort_key, rn`

	var rows []bigGroupRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return assembleBigGroups(rows), nil
}

func assembleBigGroups(rows []bigGroupRow) []model.BigGroupStat {
	var result []model.BigGroupStat
	var cur *model.BigGroupStat

	for _, row := range rows {
		if cur == nil || cur.BigGroup != row.BigGroup {
			if cur != nil {
				result = append(result, *cur)
			}
			cur = &model.BigGroupStat{
				BigGroup: row.BigGroup,
				Count:    row.BigGroupTotal,
				HomeTag:  row.BigGroup == "宪法" || row.BigGroup == "法律",
				More:     row.TotalSub > 3,
				SubTypes: make([]model.SubTypeStat, 0, 3),
			}
		}
		if row.Rank <= 3 {
			cur.SubTypes = append(cur.SubTypes, model.SubTypeStat{
				TypeID:   row.TypeID,
				TypeName: row.TypeName,
				Count:    row.Count,
			})
		}
	}
	if cur != nil {
		result = append(result, *cur)
	}

	return result
}

func (r *LawRepository) GetMetaByVersionID(ctx context.Context, versionID string) (*model.LawMeta, error) {
	var law model.LawMeta

	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId, title, lawTypeId").
		Where("versionId = ?", versionID).
		Take(&law).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &law, nil
}

// FindIDsByTitleKeywords 根据关键字查找法律 ID 列表
func (r *LawRepository) FindIDsByTitleKeywords(ctx context.Context, keywords []string) ([]string, error) {
	if len(keywords) == 0 {
		return nil, nil
	}

	conditions := make([]string, 0, len(keywords))
	args := make([]any, 0, len(keywords))
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		conditions = append(conditions, "title LIKE ?")
		args = append(args, "%"+kw+"%")
	}
	if len(conditions) == 0 {
		return nil, nil
	}

	var ids []string
	err := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("versionId").
		Where(strings.Join(conditions, " OR "), args...).
		Pluck("versionId", &ids).Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

const searchLawListOrder = `
CASE WHEN l.effectDate IS NULL OR TRIM(l.effectDate) = '' THEN 1 ELSE 0 END ASC,
l.effectDate DESC,
CASE WHEN l.publishDate IS NULL OR TRIM(l.publishDate) = '' THEN 1 ELSE 0 END ASC,
l.publishDate DESC,
l.versionId DESC
`

func (r *LawRepository) buildSearchQuery(ctx context.Context, filter LawSearchFilter) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("laws_list AS l")

	// scope 复用 types/parent types 分组逻辑，和首页 tab 语义保持一致
	switch strings.TrimSpace(filter.Scope) {
	case "laws":
		query = query.
			Joins("JOIN types t ON t.id = l.lawTypeId").
			Joins("LEFT JOIN types parent_t ON t.parent_id = parent_t.id").
			Where("COALESCE(parent_t.id, t.id) IN ?", []int{100, 101, 102})
	case "admin_regulations":
		query = query.
			Joins("JOIN types t ON t.id = l.lawTypeId").
			Joins("LEFT JOIN types parent_t ON t.parent_id = parent_t.id").
			Where("COALESCE(parent_t.id, t.id) IN ?", []int{210})
	case "judicial_interpretations":
		query = query.
			Joins("JOIN types t ON t.id = l.lawTypeId").
			Joins("LEFT JOIN types parent_t ON t.parent_id = parent_t.id").
			Where("COALESCE(parent_t.id, t.id) IN ?", []int{320, 330, 340, 350})
	case "local_laws":
		query = query.
			Joins("JOIN types t ON t.id = l.lawTypeId").
			Joins("LEFT JOIN types parent_t ON t.parent_id = parent_t.id").
			Where("COALESCE(parent_t.id, t.id) IN ?", []int{222})
	}

	if trimmedQuery := strings.TrimSpace(filter.Query); trimmedQuery != "" {
		likeQuery := "%" + trimmedQuery + "%"
		// 正文搜索暂时直接用 detailJson LIKE，先满足接口能力，后面再视数据量做索引优化
		switch strings.TrimSpace(filter.TextMode) {
		case "body":
			query = query.Where("TRIM(COALESCE(l.detailJson, '')) != '' AND l.detailJson LIKE ?", likeQuery)
		case "title_and_body":
			query = query.Where(
				"l.title LIKE ? AND TRIM(COALESCE(l.detailJson, '')) != '' AND l.detailJson LIKE ?",
				likeQuery,
				likeQuery,
			)
		case "title_or_body":
			query = query.Where(
				"(l.title LIKE ?) OR (TRIM(COALESCE(l.detailJson, '')) != '' AND l.detailJson LIKE ?)",
				likeQuery,
				likeQuery,
			)
		default:
			query = query.Where("l.title LIKE ?", likeQuery)
		}
	}

	if len(filter.AuthorityNames) > 0 {
		query = query.Where("l.authorityName IN ?", filter.AuthorityNames)
	}
	if len(filter.EffectiveStatuses) > 0 {
		query = query.Where("l.effectiveStatus IN ?", filter.EffectiveStatuses)
	}

	query = applyDateRangeFilter(query, "l.publishDate", filter.PublishDateStart, filter.PublishDateEnd)
	query = applyDateRangeFilter(query, "l.effectDate", filter.EffectDateStart, filter.EffectDateEnd)

	return query
}

func applyDateRangeFilter(query *gorm.DB, column, start, end string) *gorm.DB {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" && end == "" {
		return query
	}

	// 只要设置了日期范围，就要求该字段本身非空，避免空日期也被误命中
	query = query.Where("TRIM(COALESCE("+column+", '')) != ''")
	if start != "" {
		query = query.Where(column+" >= ?", start)
	}
	if end != "" {
		query = query.Where(column+" <= ?", end)
	}
	return query
}
