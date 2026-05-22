package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"LawHelperServer/internal/model"
)

type CommonLawRepository struct {
	db *gorm.DB
}

func NewCommonLawRepository(db *gorm.DB) *CommonLawRepository {
	return &CommonLawRepository{db: db}
}

// TablesExist 检查两个表是否都已存在
func (r *CommonLawRepository) TablesExist(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('common_law_type', 'common_laws')",
	).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count == 2, nil
}

// CreateTables 创建表（使用 GORM AutoMigrate）
func (r *CommonLawRepository) CreateTables(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(
		&model.CommonLawType{},
		&model.CommonLaw{},
	)
}

// ListAllTypes 获取所有常用法律类型（按 sort_order 排序）
func (r *CommonLawRepository) ListAllTypes(ctx context.Context) ([]model.CommonLawType, error) {
	var types []model.CommonLawType
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, id ASC").
		Find(&types).Error
	return types, err
}

// UpsertTypes 批量插入或更新类型定义
func (r *CommonLawRepository) UpsertTypes(ctx context.Context, types []model.CommonLawType) error {
	if len(types) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uuid"}},
			DoUpdates: clause.AssignmentColumns([]string{"type_id", "law_type", "law_type_display", "icon", "keywords", "sort_order", "updated_at"}),
		}).
		CreateInBatches(types, 100).Error
}

// GetTypesCount 获取各类型的法律数量
func (r *CommonLawRepository) GetTypesCount(ctx context.Context) (map[int]int64, error) {
	var results []struct {
		CommonLawTypeID int   `gorm:"column:common_law_type_id"`
		Count           int64 `gorm:"column:count"`
	}
	err := r.db.WithContext(ctx).
		Model(&model.CommonLaw{}).
		Select("common_law_type_id, COUNT(*) as count").
		Group("common_law_type_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[int]int64)
	for _, res := range results {
		counts[res.CommonLawTypeID] = res.Count
	}
	return counts, nil
}

// ClearAllMappings 清除所有映射
func (r *CommonLawRepository) ClearAllMappings(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("1 = 1").
		Delete(&model.CommonLaw{}).Error
}

// BatchInsertMappings 批量插入映射关系（忽略重复）
func (r *CommonLawRepository) BatchInsertMappings(ctx context.Context, mappings []model.CommonLaw) error {
	if len(mappings) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "common_law_type_id"}, {Name: "law_id"}},
			DoNothing: true,
		}).
		CreateInBatches(mappings, 500).Error
}

// HasMappings 检查是否有映射数据
func (r *CommonLawRepository) HasMappings(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.CommonLaw{}).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountByTypeID 根据类型ID统计法律数量
func (r *CommonLawRepository) CountByTypeID(ctx context.Context, typeID int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.CommonLaw{}).
		Where("common_law_type_id = ?", typeID).
		Count(&count).Error
	return count, err
}

// ListLawsByTypeID 根据类型ID分页查询法律列表
func (r *CommonLawRepository) ListLawsByTypeID(ctx context.Context, typeID, offset, limit int) ([]model.LawSummary, error) {
	var laws []model.LawSummary
	err := r.db.WithContext(ctx).
		Table("common_laws c").
		Select("l.versionId, l.title, l.lawTypeId, l.lawType, l.publishDate, l.effectDate, l.effectiveStatus, l.authorityName").
		Joins("JOIN laws_list l ON c.law_id = l.versionId").
		Where("c.common_law_type_id = ?", typeID).
		Order(lawListOrder).
		Offset(offset).
		Limit(limit).
		Find(&laws).Error
	return laws, err
}

// ListAllLawsByTypeID 根据类型ID查询全部法律列表
func (r *CommonLawRepository) ListAllLawsByTypeID(ctx context.Context, typeID int) ([]model.LawSummary, error) {
	var laws []model.LawSummary
	err := r.db.WithContext(ctx).
		Table("common_laws c").
		Select("l.versionId, l.title, l.lawTypeId, l.lawType, l.publishDate, l.effectDate, l.effectiveStatus, l.authorityName").
		Joins("JOIN laws_list l ON c.law_id = l.versionId").
		Where("c.common_law_type_id = ?", typeID).
		Order(lawListOrder).
		Find(&laws).Error
	return laws, err
}

// GetTypeByID 根据ID获取类型
func (r *CommonLawRepository) GetTypeByID(ctx context.Context, id int) (*model.CommonLawType, error) {
	var t model.CommonLawType
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Take(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
