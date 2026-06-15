package service

import (
	"context"
	"encoding/json"
	"log"

	"LawHelperServer/internal/model"
	"LawHelperServer/internal/repository"
)

// SyncService 负责常用法律数据和地方法律机关数据的同步
type SyncService struct {
	commonLawRepo      *repository.CommonLawRepository
	lawRepo            *repository.LawRepository
	localAuthorityRepo *repository.LocalAuthorityRepository
}

func NewSyncService(commonLawRepo *repository.CommonLawRepository, lawRepo *repository.LawRepository, localAuthorityRepo *repository.LocalAuthorityRepository) *SyncService {
	return &SyncService{
		commonLawRepo:      commonLawRepo,
		lawRepo:            lawRepo,
		localAuthorityRepo: localAuthorityRepo,
	}
}

// SyncCommonLaws 同步常用法律数据（启动时调用）
func (s *SyncService) SyncCommonLaws(ctx context.Context) error {
	// 1. 检查表是否存在
	exists, err := s.commonLawRepo.TablesExist(ctx)
	if err != nil {
		return err
	}

	if !exists {
		// 2. 创建表
		if err := s.commonLawRepo.CreateTables(ctx); err != nil {
			return err
		}
		log.Println("Created common_law_type and common_laws tables")
	}

	// 3. 同步类型定义
	if err := s.syncTypes(ctx); err != nil {
		return err
	}

	// 4. 检查是否需要同步映射
	needSync, err := s.needSyncMappings(ctx)
	if err != nil {
		return err
	}

	if needSync {
		log.Println("Syncing common_laws mappings...")
		if err := s.syncMappings(ctx); err != nil {
			return err
		}
		log.Println("Common laws sync completed")
	}

	return nil
}

// syncTypes 同步类型定义
func (s *SyncService) syncTypes(ctx context.Context) error {
	types := make([]model.CommonLawType, 0, len(commonLawDefs))

	for i, def := range commonLawDefs {
		keywordsJSON, err := json.Marshal(def.Keywords)
		if err != nil {
			return err
		}

		types = append(types, model.CommonLawType{
			UUID:           def.UUID,
			TypeID:         def.TypeID,
			LawType:        def.LawType,
			LawTypeDisplay: def.LawTypeDisplay,
			Icon:           def.Icon,
			Keywords:       string(keywordsJSON),
			SortOrder:      i,
		})
	}

	return s.commonLawRepo.UpsertTypes(ctx, types)
}

// needSyncMappings 判断是否需要同步映射
func (s *SyncService) needSyncMappings(ctx context.Context) (bool, error) {
	// 检查是否有映射数据
	hasMappings, err := s.commonLawRepo.HasMappings(ctx)
	if err != nil {
		return false, err
	}
	return !hasMappings, nil
}

// syncMappings 同步法律映射关系
func (s *SyncService) syncMappings(ctx context.Context) error {
	types, err := s.commonLawRepo.ListAllTypes(ctx)
	if err != nil {
		return err
	}

	// 先清除所有映射
	if err := s.commonLawRepo.ClearAllMappings(ctx); err != nil {
		return err
	}

	// 按类型批量同步
	for _, t := range types {
		var keywords []string
		if err := json.Unmarshal([]byte(t.Keywords), &keywords); err != nil {
			return err
		}

		// 查询匹配的法律 ID
		lawIDs, err := s.lawRepo.FindIDsByTitleKeywords(ctx, keywords)
		if err != nil {
			return err
		}

		// 批量插入映射
		mappings := make([]model.CommonLaw, 0, len(lawIDs))
		for _, lawID := range lawIDs {
			mappings = append(mappings, model.CommonLaw{
				CommonLawTypeID: t.ID,
				LawID:           lawID,
			})
		}

		if len(mappings) > 0 {
			if err := s.commonLawRepo.BatchInsertMappings(ctx, mappings); err != nil {
				return err
			}
			log.Printf("Synced %d laws for type: %s", len(mappings), t.LawTypeDisplay)
		}
	}

	return nil
}

// SyncLocalAuthorities 同步地方法律制定机关数据（启动时调用）
// 从 laws_list 提取地方法律的 authorityName，按省份和城市归属写入 local_authority_list 表
func (s *SyncService) SyncLocalAuthorities(ctx context.Context) error {
	// 1. 检查表是否存在，不存在则创建
	exists, err := s.localAuthorityRepo.TableExists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		if err := s.localAuthorityRepo.CreateTable(ctx); err != nil {
			return err
		}
		log.Println("Created local_authority_list table")
	}

	// 2. 如果已有数据则跳过
	hasData, err := s.localAuthorityRepo.HasData(ctx)
	if err != nil {
		return err
	}
	if hasData {
		return nil
	}

	// 3. 从 laws_list 聚合地方法律的制定机关数据
	rows, err := s.localAuthorityRepo.QueryAuthorityStats(ctx, LocalLawTypeIDs)
	if err != nil {
		return err
	}

	// 4. 对每个 authorityName 解析省/市归属
	authorities := make([]model.LocalAuthority, 0, len(rows))
	for _, row := range rows {
		province, city := resolveLocation(row.AuthorityName)
		authorities = append(authorities, model.LocalAuthority{
			AuthorityName: row.AuthorityName,
			Province:      province,
			City:          city,
			LawCount:      row.LawCount,
		})
	}

	// 5. 写入表
	if err := s.localAuthorityRepo.ReplaceAll(ctx, authorities); err != nil {
		return err
	}

	log.Printf("Synced %d local authorities", len(authorities))
	return nil
}
