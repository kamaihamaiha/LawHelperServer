package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"LawHelperServer/internal/model"
)

type TypeRepository struct {
	db *gorm.DB
}

func NewTypeRepository(db *gorm.DB) *TypeRepository {
	return &TypeRepository{db: db}
}

func (r *TypeRepository) ListConcreteTypesWithLawCount(ctx context.Context) ([]model.TypeWithCount, error) {
	var types []model.TypeWithCount

	subQuery := r.db.WithContext(ctx).
		Model(&model.LawList{}).
		Select("lawTypeId AS type_id, COUNT(*) AS law_count").
		Group("lawTypeId")

	err := r.db.WithContext(ctx).
		Table("types AS t").
		Select("t.id, t.name, t.parent_id, counts.law_count").
		Joins("JOIN (?) AS counts ON counts.type_id = t.id", subQuery).
		Order("t.id ASC").
		Scan(&types).Error
	if err != nil {
		return nil, err
	}

	return types, nil
}

func (r *TypeRepository) ListByIDs(ctx context.Context, ids []int) ([]model.Type, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var types []model.Type
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&types).Error
	if err != nil {
		return nil, err
	}

	return types, nil
}

func (r *TypeRepository) GetByID(ctx context.Context, id int) (*model.Type, error) {
	var lawType model.Type

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Take(&lawType).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &lawType, nil
}
