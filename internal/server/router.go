package server

import (
	"github.com/gin-gonic/gin"

	"LawHelperServer/internal/handler"
)

func NewRouter(lawHandler *handler.LawHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/healthz", lawHandler.Healthz)

	api := router.Group("/api/v1")
	api.GET("/types/previews", lawHandler.ListTypePreviews)
	api.GET("/types/:typeId/laws", lawHandler.ListLawsByType) //获取某个分类的全部法律list
	api.GET("/laws/big-groups", lawHandler.ListBigGroupStats)
	api.POST("/laws/search", lawHandler.SearchLaws)
	api.GET("/laws/:versionId/parsed", lawHandler.GetParsedLaw)                  //获取法律详情
	api.GET("/home/laws", lawHandler.GetHomeLaws)                                //首页法律接口: 返回不同类型的法律
	api.GET("/new-laws", lawHandler.ListNewLaws)                                 //新法速递列表, 支持分页
	api.GET("/common-laws/:typeId/laws", lawHandler.ListCommonLawsByType)        //获取某个常用法律类型的全部法律list
	api.GET("/admin-regulations", lawHandler.ListAdminRegulations)               //行政法规列表, 一次性返回全部简介
	api.GET("/judicial-interpretations", lawHandler.ListJudicialInterpretations) //司法解释列表, 一次性返回全部简介
	api.GET("/local-laws", lawHandler.ListLocalLaws)                             //地方法律列表, 支持分页, 默认 50 条
	api.GET("/local-laws/authorities", lawHandler.ListLocalAuthorities)         //地方法律制定机关列表, 一次返回全部, 供三级联动选择

	return router
}
