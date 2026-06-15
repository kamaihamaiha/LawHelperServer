package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"LawHelperServer/internal/model"
	"LawHelperServer/internal/repository"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
	previewLimit    = 20
)

var (
	ErrTypeNotFound          = errors.New("type not found")
	ErrLawNotFound           = errors.New("law not found")
	ErrCommonLawTypeNotFound = errors.New("common law type not found")
)

type LawService struct {
	typeRepo           *repository.TypeRepository
	lawRepo            *repository.LawRepository
	parsedLawRepo      *repository.ParsedLawRepository
	commonLawRepo      *repository.CommonLawRepository
	localAuthorityRepo *repository.LocalAuthorityRepository
}

type TypePreview struct {
	TypeID   int                `json:"typeId"`
	TypeName string             `json:"typeName"`
	ParentID *int               `json:"parentId"`
	Total    int64              `json:"total"`
	Items    []model.LawSummary `json:"items"`
}

type PaginatedLawList struct {
	Type       TypeInfo           `json:"type"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"totalPages"`
	Items      []model.LawSummary `json:"items"`
}

type TypeInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID *int   `json:"parentId"`
}

type CommonLawTypeInfo struct {
	ID             int    `json:"id"`
	LawType        string `json:"lawType"`
	LawTypeDisplay string `json:"lawTypeDisplay"`
	Icon           string `json:"icon"`
}

type PaginatedCommonLawList struct {
	Type       CommonLawTypeInfo  `json:"type"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"totalPages"`
	Items      []model.LawSummary `json:"items"`
}

type PaginatedNewLawList struct {
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"totalPages"`
	Items      []model.LawSummary `json:"items"`
}

const (
	SearchScopeLaws                    = "laws"                     //法律
	SearchScopeAdminRegulations        = "admin_regulations"        //行政法规
	SearchScopeJudicialInterpretations = "judicial_interpretations" //司法解释
	SearchScopeLocalLaws               = "local_laws"               //地方法律
)

const (
	SearchTextModeTitle        = "title"
	SearchTextModeBody         = "body"
	SearchTextModeTitleAndBody = "title_and_body"
	SearchTextModeTitleOrBody  = "title_or_body"
)

type LawSearchRequest struct {
	// 这份结构既是 handler -> service 的输入，也限定了 repository 可见的搜索维度
	Scope             string   `json:"scope"`
	Query             string   `json:"query"`
	TextMode          string   `json:"textMode"`
	AuthorityNames    []string `json:"authorityNames"`
	EffectiveStatuses []int    `json:"effectiveStatuses"`
	PublishDateStart  string   `json:"publishDateStart"`
	PublishDateEnd    string   `json:"publishDateEnd"`
	EffectDateStart   string   `json:"effectDateStart"`
	EffectDateEnd     string   `json:"effectDateEnd"`
	Page              int      `json:"page"`
	PageSize          int      `json:"pageSize"`
}

// 类型ID分组
var (
	// 行政法规类型ID
	AdminRegulationTypeIDs = []int{210, 215}
	// 司法解释类型ID
	JudicialInterpretationTypeIDs = []int{320, 330, 340, 350}
	// 地方法律类型ID
	LocalLawTypeIDs = []int{222, 230, 260, 270, 290, 295, 300, 305, 310}
)

type ParsedLawDetail struct {
	VersionID string           `json:"versionId"`
	Title     string           `json:"title"`
	Available bool             `json:"available"`
	Content   *json.RawMessage `json:"content"`
}

func NewLawService(typeRepo *repository.TypeRepository, lawRepo *repository.LawRepository, parsedLawRepo *repository.ParsedLawRepository, commonLawRepo *repository.CommonLawRepository, localAuthorityRepo *repository.LocalAuthorityRepository) *LawService {
	return &LawService{
		typeRepo:           typeRepo,
		lawRepo:            lawRepo,
		parsedLawRepo:      parsedLawRepo,
		commonLawRepo:      commonLawRepo,
		localAuthorityRepo: localAuthorityRepo,
	}
}

func (s *LawService) ListTypePreviews(ctx context.Context) ([]TypePreview, error) {
	lawTypes, err := s.typeRepo.ListConcreteTypesWithLawCount(ctx)
	if err != nil {
		return nil, err
	}

	previews := make([]TypePreview, 0, len(lawTypes))
	for _, lawType := range lawTypes {
		items, err := s.lawRepo.ListByType(ctx, lawType.ID, 0, previewLimit)
		if err != nil {
			return nil, err
		}

		previews = append(previews, TypePreview{
			TypeID:   lawType.ID,
			TypeName: lawType.Name,
			ParentID: lawType.ParentID,
			Total:    lawType.LawCount,
			Items:    items,
		})
	}

	return previews, nil
}

func (s *LawService) ListLawsByType(ctx context.Context, typeID int) (*PaginatedLawList, error) {
	lawType, err := s.typeRepo.GetByID(ctx, typeID)
	if err != nil {
		return nil, err
	}
	if lawType == nil {
		return nil, ErrTypeNotFound
	}

	total, err := s.lawRepo.CountByType(ctx, typeID)
	if err != nil {
		return nil, err
	}

	items, err := s.lawRepo.ListAllByType(ctx, typeID)
	if err != nil {
		return nil, err
	}

	pageSize := len(items)
	return &PaginatedLawList{
		Type: TypeInfo{
			ID:       lawType.ID,
			Name:     lawType.Name,
			ParentID: lawType.ParentID,
		},
		Page:       defaultPage,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
		Items:      items,
	}, nil
}

func (s *LawService) ListCommonLawsByType(ctx context.Context, typeID int) (*PaginatedCommonLawList, error) {
	commonLawType, err := s.commonLawRepo.GetTypeByID(ctx, typeID)
	if err != nil {
		return nil, err
	}
	if commonLawType == nil {
		return nil, ErrCommonLawTypeNotFound
	}

	total, err := s.commonLawRepo.CountByTypeID(ctx, typeID)
	if err != nil {
		return nil, err
	}

	items, err := s.commonLawRepo.ListAllLawsByTypeID(ctx, typeID)
	if err != nil {
		return nil, err
	}

	pageSize := len(items)
	return &PaginatedCommonLawList{
		Type: CommonLawTypeInfo{
			ID:             commonLawType.ID,
			LawType:        commonLawType.LawType,
			LawTypeDisplay: commonLawType.LawTypeDisplay,
			Icon:           commonLawType.Icon,
		},
		Page:       defaultPage,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
		Items:      items,
	}, nil
}

func (s *LawService) ListNewLaws(ctx context.Context, page, pageSize int) (*PaginatedNewLawList, error) {
	page, pageSize = normalizePagination(page, pageSize)

	now := time.Now()
	today := now.Format("2006-01-02")
	cutoff := now.AddDate(0, -6, 0).Format("2006-01-02")

	total, err := s.lawRepo.CountNewLaws(ctx, cutoff, today)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	items, err := s.lawRepo.ListNewLawsPaginated(ctx, cutoff, today, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return &PaginatedNewLawList{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
		Items:      items,
	}, nil
}

func (s *LawService) ListAdminRegulations(ctx context.Context) (*PaginatedNewLawList, error) {
	return s.listAllLawsByTypeIDs(ctx, AdminRegulationTypeIDs)
}

func (s *LawService) ListAdminRegulationsPage(ctx context.Context, page, pageSize int) (*PaginatedNewLawList, error) {
	return s.listLawsByTypeIDs(ctx, AdminRegulationTypeIDs, page, pageSize)
}

func (s *LawService) ListJudicialInterpretations(ctx context.Context) (*PaginatedNewLawList, error) {
	return s.listAllLawsByTypeIDs(ctx, JudicialInterpretationTypeIDs)
}

func (s *LawService) ListJudicialInterpretationsPage(ctx context.Context, page, pageSize int) (*PaginatedNewLawList, error) {
	return s.listLawsByTypeIDs(ctx, JudicialInterpretationTypeIDs, page, pageSize)
}

func (s *LawService) ListLocalLaws(ctx context.Context, page, pageSize int) (*PaginatedNewLawList, error) {
	return s.listLawsByTypeIDs(ctx, LocalLawTypeIDs, page, pageSize)
}

func (s *LawService) listLawsByTypeIDs(ctx context.Context, typeIDs []int, page, pageSize int) (*PaginatedNewLawList, error) {
	page, pageSize = normalizePagination(page, pageSize)

	total, err := s.lawRepo.CountByTypeIDs(ctx, typeIDs)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	items, err := s.lawRepo.ListByTypeIDs(ctx, typeIDs, offset, pageSize)
	if err != nil {
		return nil, err
	}

	return &PaginatedNewLawList{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
		Items:      items,
	}, nil
}

func (s *LawService) listAllLawsByTypeIDs(ctx context.Context, typeIDs []int) (*PaginatedNewLawList, error) {
	total, err := s.lawRepo.CountByTypeIDs(ctx, typeIDs)
	if err != nil {
		return nil, err
	}

	items, err := s.lawRepo.ListAllByTypeIDs(ctx, typeIDs)
	if err != nil {
		return nil, err
	}

	pageSize := len(items)
	return &PaginatedNewLawList{
		Page:       defaultPage,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
		Items:      items,
	}, nil
}

func (s *LawService) ListBigGroupStats(ctx context.Context) ([]model.BigGroupStat, error) {
	return s.lawRepo.ListBigGroupStats(ctx)
}

func (s *LawService) GetParsedLaw(ctx context.Context, versionID string) (*ParsedLawDetail, error) {
	versionID = strings.TrimSpace(versionID)

	lawMeta, err := s.lawRepo.GetMetaByVersionID(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if lawMeta == nil {
		return nil, ErrLawNotFound
	}

	raw, err := s.parsedLawRepo.GetByVersionID(ctx, versionID, lawMeta.LawTypeID)
	if err != nil {
		if errors.Is(err, repository.ErrParsedLawNotFound) {
			return &ParsedLawDetail{
				VersionID: lawMeta.VersionID,
				Title:     lawMeta.Title,
				Available: false,
				Content:   nil,
			}, nil
		}
		return nil, err
	}

	return &ParsedLawDetail{
		VersionID: lawMeta.VersionID,
		Title:     lawMeta.Title,
		Available: true,
		Content:   &raw,
	}, nil
}

func (s *LawService) SearchLaws(ctx context.Context, req LawSearchRequest) (*PaginatedNewLawList, error) {
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	req.Page = page
	req.PageSize = pageSize
	req.Scope = strings.TrimSpace(req.Scope)
	req.Query = strings.TrimSpace(req.Query)
	req.TextMode = strings.TrimSpace(req.TextMode)
	if req.TextMode == "" {
		req.TextMode = SearchTextModeTitle
	}

	// 先 count 再查列表，保持和现有分页接口一致的返回结构
	total, err := s.lawRepo.CountSearchLaws(ctx, repository.LawSearchFilter{
		Scope:             req.Scope,
		Query:             req.Query,
		TextMode:          req.TextMode,
		AuthorityNames:    req.AuthorityNames,
		EffectiveStatuses: req.EffectiveStatuses,
		PublishDateStart:  req.PublishDateStart,
		PublishDateEnd:    req.PublishDateEnd,
		EffectDateStart:   req.EffectDateStart,
		EffectDateEnd:     req.EffectDateEnd,
	})
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	items, err := s.lawRepo.SearchLaws(ctx, repository.LawSearchFilter{
		Scope:             req.Scope,
		Query:             req.Query,
		TextMode:          req.TextMode,
		AuthorityNames:    req.AuthorityNames,
		EffectiveStatuses: req.EffectiveStatuses,
		PublishDateStart:  req.PublishDateStart,
		PublishDateEnd:    req.PublishDateEnd,
		EffectDateStart:   req.EffectDateStart,
		EffectDateEnd:     req.EffectDateEnd,
		Offset:            offset,
		Limit:             pageSize,
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedNewLawList{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages(total, pageSize),
		Items:      items,
	}, nil
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

func totalPages(total int64, pageSize int) int {
	if total == 0 {
		return 0
	}

	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

// ListLocalAuthorities 返回全部地方法律制定机关（按省/市分组），供客户端三级联动选择
func (s *LawService) ListLocalAuthorities(ctx context.Context) ([]model.LocalAuthority, error) {
	return s.localAuthorityRepo.ListAll(ctx)
}
