package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"LawHelperServer/internal/response"
	"LawHelperServer/internal/service"
)

type LawHandler struct {
	lawService *service.LawService
}

func NewLawHandler(lawService *service.LawService) *LawHandler {
	return &LawHandler{lawService: lawService}
}

func (h *LawHandler) Healthz(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
	})
}

func (h *LawHandler) ListTypePreviews(c *gin.Context) {
	previews, err := h.lawService.ListTypePreviews(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询分类预览失败")
		return
	}

	response.Success(c, previews)
}

func (h *LawHandler) ListLawsByType(c *gin.Context) {
	typeID, err := strconv.Atoi(c.Param("typeId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "typeId 必须是整数")
		return
	}

	result, err := h.lawService.ListLawsByType(c.Request.Context(), typeID)
	if err != nil {
		if errors.Is(err, service.ErrTypeNotFound) {
			response.Error(c, http.StatusNotFound, "分类不存在")
			return
		}

		response.Error(c, http.StatusInternalServerError, "查询分类列表失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) GetHomeLaws(c *gin.Context) {
	result, err := h.lawService.GetHomeLaws(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询首页法律数据失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) ListNewLaws(c *gin.Context) {
	page, err := strconv.Atoi(defaultQuery(c, "page", "1"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "page 必须是整数")
		return
	}

	pageSize, err := strconv.Atoi(defaultQuery(c, "pageSize", "20"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "pageSize 必须是整数")
		return
	}

	result, err := h.lawService.ListNewLaws(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询新法速递列表失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) ListCommonLawsByType(c *gin.Context) {
	typeID, err := strconv.Atoi(c.Param("typeId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "typeId 必须是整数")
		return
	}

	result, err := h.lawService.ListCommonLawsByType(c.Request.Context(), typeID)
	if err != nil {
		if errors.Is(err, service.ErrCommonLawTypeNotFound) {
			response.Error(c, http.StatusNotFound, "常用法律类型不存在")
			return
		}

		response.Error(c, http.StatusInternalServerError, "查询常用法律列表失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) ListAdminRegulations(c *gin.Context) {
	var result any
	var err error
	if hasPaginationQuery(c) {
		page, pageSize, message, ok := parsePaginationQuery(c, "20")
		if !ok {
			response.Error(c, http.StatusBadRequest, message)
			return
		}
		result, err = h.lawService.ListAdminRegulationsPage(c.Request.Context(), page, pageSize)
	} else {
		result, err = h.lawService.ListAdminRegulations(c.Request.Context())
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询行政法规列表失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) ListJudicialInterpretations(c *gin.Context) {
	var result any
	var err error
	if hasPaginationQuery(c) {
		page, pageSize, message, ok := parsePaginationQuery(c, "20")
		if !ok {
			response.Error(c, http.StatusBadRequest, message)
			return
		}
		result, err = h.lawService.ListJudicialInterpretationsPage(c.Request.Context(), page, pageSize)
	} else {
		result, err = h.lawService.ListJudicialInterpretations(c.Request.Context())
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询司法解释列表失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) ListLocalLaws(c *gin.Context) {
	page, err := strconv.Atoi(defaultQuery(c, "page", "1"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "page 必须是整数")
		return
	}

	pageSize, err := strconv.Atoi(defaultQuery(c, "pageSize", "50"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "pageSize 必须是整数")
		return
	}

	result, err := h.lawService.ListLocalLaws(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询地方法律列表失败")
		return
	}

	response.Success(c, result)
}

func (h *LawHandler) ListBigGroupStats(c *gin.Context) {
	stats, err := h.lawService.ListBigGroupStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询大分类统计失败")
		return
	}

	response.Success(c, stats)
}

func (h *LawHandler) GetParsedLaw(c *gin.Context) {
	versionID := strings.TrimSpace(c.Param("versionId"))
	if versionID == "" {
		response.Error(c, http.StatusBadRequest, "versionId 不能为空")
		return
	}

	result, err := h.lawService.GetParsedLaw(c.Request.Context(), versionID)
	if err != nil {
		if errors.Is(err, service.ErrLawNotFound) {
			response.Error(c, http.StatusNotFound, "法律不存在")
			return
		}

		response.Error(c, http.StatusInternalServerError, "读取法律详情失败")
		return
	}

	if !result.Available {
		response.SuccessWithMessage(c, "暂无解析数据", result)
		return
	}

	response.Success(c, result)
}

func defaultQuery(c *gin.Context, key, fallback string) string {
	if value := strings.TrimSpace(c.Query(key)); value != "" {
		return value
	}

	return fallback
}

type searchLawsRequest struct {
	// 字段命名直接对齐 Android 端请求体，减少前后端映射成本
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

func (h *LawHandler) SearchLaws(c *gin.Context) {
	var req searchLawsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	if !isValidSearchScope(req.Scope) {
		response.Error(c, http.StatusBadRequest, "scope 不合法")
		return
	}
	if !isValidTextMode(req.TextMode) {
		response.Error(c, http.StatusBadRequest, "textMode 不合法")
		return
	}
	if !isValidDateString(req.PublishDateStart) || !isValidDateString(req.PublishDateEnd) ||
		!isValidDateString(req.EffectDateStart) || !isValidDateString(req.EffectDateEnd) {
		response.Error(c, http.StatusBadRequest, "日期必须是 yyyy-MM-dd 格式")
		return
	}

	// Handler 只负责参数校验和转换，具体过滤逻辑下沉到 service/repository
	result, err := h.lawService.SearchLaws(c.Request.Context(), service.LawSearchRequest{
		Scope:             strings.TrimSpace(req.Scope),
		Query:             strings.TrimSpace(req.Query),
		TextMode:          strings.TrimSpace(req.TextMode),
		AuthorityNames:    normalizeStringSlice(req.AuthorityNames),
		EffectiveStatuses: normalizeIntSlice(req.EffectiveStatuses),
		PublishDateStart:  strings.TrimSpace(req.PublishDateStart),
		PublishDateEnd:    strings.TrimSpace(req.PublishDateEnd),
		EffectDateStart:   strings.TrimSpace(req.EffectDateStart),
		EffectDateEnd:     strings.TrimSpace(req.EffectDateEnd),
		Page:              req.Page,
		PageSize:          req.PageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "按条件搜索法律失败")
		return
	}

	response.Success(c, result)
}

func hasPaginationQuery(c *gin.Context) bool {
	return strings.TrimSpace(c.Query("page")) != "" || strings.TrimSpace(c.Query("pageSize")) != ""
}

func parsePaginationQuery(c *gin.Context, defaultPageSize string) (int, int, string, bool) {
	page, err := strconv.Atoi(defaultQuery(c, "page", "1"))
	if err != nil {
		return 0, 0, "page 必须是整数", false
	}

	pageSize, err := strconv.Atoi(defaultQuery(c, "pageSize", defaultPageSize))
	if err != nil {
		return 0, 0, "pageSize 必须是整数", false
	}

	return page, pageSize, "", true
}

func isValidSearchScope(scope string) bool {
	switch strings.TrimSpace(scope) {
	case "", service.SearchScopeLaws, service.SearchScopeAdminRegulations, service.SearchScopeJudicialInterpretations, service.SearchScopeLocalLaws:
		return true
	default:
		return false
	}
}

func isValidTextMode(textMode string) bool {
	switch strings.TrimSpace(textMode) {
	case "", service.SearchTextModeTitle, service.SearchTextModeBody, service.SearchTextModeTitleAndBody, service.SearchTextModeTitleOrBody:
		return true
	default:
		return false
	}
}

func isValidDateString(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}

	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func normalizeStringSlice(values []string) []string {
	// 去重 + 去空，避免 repository 层处理重复条件
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeIntSlice(values []int) []int {
	// 状态值只保留正整数，非法值在这里提前裁掉
	result := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
