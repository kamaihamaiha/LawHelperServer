package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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

	result, err := h.lawService.ListLawsByType(c.Request.Context(), typeID, page, pageSize)
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
