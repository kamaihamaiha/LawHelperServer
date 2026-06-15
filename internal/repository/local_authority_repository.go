package repository

import (
	"context"

	"gorm.io/gorm"

	"LawHelperServer/internal/model"
)

type LocalAuthorityRepository struct {
	db *gorm.DB
}

func NewLocalAuthorityRepository(db *gorm.DB) *LocalAuthorityRepository {
	return &LocalAuthorityRepository{db: db}
}

// CreateTable 使用 GORM AutoMigrate 创建 local_authority_list 表
func (r *LocalAuthorityRepository) CreateTable(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&model.LocalAuthority{})
}

// TableExists 检查 local_authority_list 表是否已存在
func (r *LocalAuthorityRepository) TableExists(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = 'local_authority_list'",
	).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// HasData 检查表中是否已有数据
func (r *LocalAuthorityRepository) HasData(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.LocalAuthority{}).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ReplaceAll 在事务中替换全部数据：先清空再批量插入
func (r *LocalAuthorityRepository) ReplaceAll(ctx context.Context, authorities []model.LocalAuthority) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先清空旧数据
		if err := tx.Where("1 = 1").Delete(&model.LocalAuthority{}).Error; err != nil {
			return err
		}
		// 批量插入新数据
		if len(authorities) > 0 {
			return tx.CreateInBatches(authorities, 100).Error
		}
		return nil
	})
}

// ListAll 查询全部制定机关，按省/市/机关名排序，方便客户端三级分组
func (r *LocalAuthorityRepository) ListAll(ctx context.Context) ([]model.LocalAuthority, error) {
	var authorities []model.LocalAuthority
	err := r.db.WithContext(ctx).
		Order("province ASC, city ASC, authorityName ASC").
		Find(&authorities).Error
	return authorities, err
}

// AuthorityStat 原始聚合结果：每个 authorityName 及其地方法律数量
type AuthorityStat struct {
	AuthorityName string
	LawCount      int64
}

// QueryAuthorityStats 从 laws_list 聚合地方法律的制定机关统计数据
// localLawTypeIDs: 地方法律的类型 ID 列表
func (r *LocalAuthorityRepository) QueryAuthorityStats(ctx context.Context, localLawTypeIDs []int) ([]AuthorityStat, error) {
	var rows []AuthorityStat
	err := r.db.WithContext(ctx).
		Table("laws_list").
		Select("authorityName, COUNT(*) as law_count").
		Where("lawTypeId IN ?", localLawTypeIDs).
		Where("TRIM(COALESCE(authorityName, '')) != ''").
		Group("authorityName").
		Order("law_count DESC").
		Find(&rows).Error
	return rows, err
}
