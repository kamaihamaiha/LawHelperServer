package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"LawHelperServer/internal/database"
	"LawHelperServer/internal/handler"
	"LawHelperServer/internal/repository"
	"LawHelperServer/internal/server"
	"LawHelperServer/internal/service"
)

type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func TestListTypePreviewsReturnsConcreteTypesWithAtMostTwentyItems(t *testing.T) {
	router := newTestRouter(t, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/types/previews", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[[]service.TypePreview]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	if resp.Code != 0 {
		t.Fatalf("unexpected business code: got %d want 0", resp.Code)
	}

	if got := len(resp.Data); got != 3 {
		t.Fatalf("unexpected preview type count: got %d want 3", got)
	}

	typeIDs := []int{resp.Data[0].TypeID, resp.Data[1].TypeID, resp.Data[2].TypeID}
	wantTypeIDs := []int{110, 120, 230}
	assertIntSlice(t, typeIDs, wantTypeIDs)

	if got := len(resp.Data[2].Items); got != 20 {
		t.Fatalf("unexpected preview item count for type 230: got %d want 20", got)
	}

	if got := resp.Data[2].Total; got != 22 {
		t.Fatalf("unexpected total for type 230: got %d want 22", got)
	}

	if got := resp.Data[2].Items[0].VersionID; got != "law-230-22" {
		t.Fatalf("unexpected first preview item: got %s want %s", got, "law-230-22")
	}
}

func TestListLawsByTypeSupportsPagination(t *testing.T) {
	router := newTestRouter(t, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/types/230/laws?page=2&pageSize=5", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[service.PaginatedLawList]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	if resp.Data.Page != 2 || resp.Data.PageSize != 5 {
		t.Fatalf("unexpected pagination metadata: got page=%d pageSize=%d", resp.Data.Page, resp.Data.PageSize)
	}

	if resp.Data.Total != 22 || resp.Data.TotalPages != 5 {
		t.Fatalf("unexpected total metadata: got total=%d totalPages=%d", resp.Data.Total, resp.Data.TotalPages)
	}

	if got := len(resp.Data.Items); got != 5 {
		t.Fatalf("unexpected item count: got %d want 5", got)
	}

	if got := resp.Data.Items[0].VersionID; got != "law-230-17" {
		t.Fatalf("unexpected first item on page 2: got %s want %s", got, "law-230-17")
	}
}

func TestListLawsByTypeSortsEmptyEffectDateLast(t *testing.T) {
	router := newTestRouter(t, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/types/120/laws", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[service.PaginatedLawList]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	gotOrder := []string{
		resp.Data.Items[0].VersionID,
		resp.Data.Items[1].VersionID,
		resp.Data.Items[2].VersionID,
	}
	wantOrder := []string{"law-120-b", "law-120-c", "law-120-a"}
	assertStringSlice(t, gotOrder, wantOrder)
}

func TestGetParsedLawReturnsJSONWhenFileExists(t *testing.T) {
	detailDir := t.TempDir()
	writeTypeJSONFile(t, detailDir, 110, "law-parsed-1", []byte(`{"chapters":[{"id":1}],"title":"解析版法律"}`))
	router := newTestRouter(t, detailDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/laws/law-parsed-1/parsed", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[service.ParsedLawDetail]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	if !resp.Data.Available || resp.Data.Content == nil {
		t.Fatalf("expected parsed content to be available")
	}

	var content map[string]any
	decodeJSON(t, *resp.Data.Content, &content)

	if got := content["title"]; got != "解析版法律" {
		t.Fatalf("unexpected parsed title: got %v want %v", got, "解析版法律")
	}
}

func TestGetParsedLawFallsBackToLegacyFlatPath(t *testing.T) {
	detailDir := t.TempDir()
	writeFlatJSONFile(t, detailDir, "law-parsed-1", []byte(`{"title":"旧目录解析版法律"}`))
	router := newTestRouter(t, detailDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/laws/law-parsed-1/parsed", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[service.ParsedLawDetail]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	if !resp.Data.Available || resp.Data.Content == nil {
		t.Fatalf("expected parsed content to be available")
	}

	var content map[string]any
	decodeJSON(t, *resp.Data.Content, &content)

	if got := content["title"]; got != "旧目录解析版法律" {
		t.Fatalf("unexpected parsed title: got %v want %v", got, "旧目录解析版法律")
	}
}

func TestGetParsedLawReturnsPlaceholderWhenFileMissing(t *testing.T) {
	router := newTestRouter(t, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/laws/law-parsed-missing/parsed", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[service.ParsedLawDetail]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	if resp.Message != "暂无解析数据" {
		t.Fatalf("unexpected message: got %s want %s", resp.Message, "暂无解析数据")
	}

	if resp.Data.Available {
		t.Fatalf("expected parsed content to be unavailable")
	}

	if resp.Data.Content != nil {
		t.Fatalf("expected parsed content to be nil when file is missing")
	}
}

func TestGetHomeLawsReturnsThreeSections(t *testing.T) {
	router := newTestRouter(t, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/home/laws", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusOK)
	}

	var resp envelope[service.HomeLawsResponse]
	decodeJSON(t, recorder.Body.Bytes(), &resp)

	if resp.Data.NewLawExpress.SectionTitle != "新法速递" {
		t.Fatalf("unexpected new law section title: got %s", resp.Data.NewLawExpress.SectionTitle)
	}
	if resp.Data.LawCategories.SectionTitle != "法律概览" {
		t.Fatalf("unexpected law categories section title: got %s", resp.Data.LawCategories.SectionTitle)
	}
	if resp.Data.CommonLaws.SectionTitle != "常用法律" {
		t.Fatalf("unexpected common laws section title: got %s", resp.Data.CommonLaws.SectionTitle)
	}

	newItems := resp.Data.NewLawExpress.Items
	if len(newItems) == 0 {
		t.Fatalf("expected at least one new law item")
	}
	if len(newItems) > 6 {
		t.Fatalf("new law items exceed limit: got %d want <=6", len(newItems))
	}

	foundRecent := false
	for _, item := range newItems {
		if item.UUID == "law-new-recent" {
			foundRecent = true
			if item.LawTypeDisplay != "最新发布法律" {
				t.Fatalf("unexpected display name for recent law: got %s", item.LawTypeDisplay)
			}
		}
	}
	if !foundRecent {
		t.Fatalf("expected law-new-recent in new law express section")
	}

	wantBgColors := []string{"#BF4A90D9", "#BF5C7A9E", "#BF7A5C9E"}
	for i, item := range newItems {
		if got, want := item.BgColor, wantBgColors[i%len(wantBgColors)]; got != want {
			t.Fatalf("unexpected bgColor at index %d: got %s want %s", i, got, want)
		}
	}

	categoryItems := resp.Data.LawCategories.Items
	if got := len(categoryItems); got != 8 {
		t.Fatalf("unexpected category count: got %d want 8", got)
	}

	wantCategoryOrder := []struct {
		LawType string
		Display string
		Icon    string
	}{
		{"constitution", "宪法", "account_balance"},
		{"constitution_related", "宪法相关法", "menu_book"},
		{"criminal_law", "刑法", "security"},
		{"civil_commercial_law", "民法商法", "groups"},
		{"procedural_law", "诉讼与非诉讼法", "gavel"},
		{"administrative_law", "行政法", "business_center"},
		{"economic_law", "经济法", "account_balance_wallet"},
		{"social_law", "社会法", "favorite"},
	}
	for i, want := range wantCategoryOrder {
		got := categoryItems[i]
		if got.LawType != want.LawType || got.LawTypeDisplay != want.Display || got.Icon != want.Icon {
			t.Fatalf("unexpected category at index %d: got %+v want %+v", i, got, want)
		}
	}

	commonItems := resp.Data.CommonLaws.Items
	if got := len(commonItems); got != 7 {
		t.Fatalf("unexpected common law count: got %d want 7", got)
	}

	countByLawType := make(map[string]int, len(commonItems))
	for _, item := range commonItems {
		countByLawType[item.LawType] = item.Count
	}
	if got := countByLawType["marriage_family"]; got < 1 {
		t.Fatalf("expected marriage_family count >= 1 from seeded title, got %d", got)
	}
	if got := countByLawType["labor_personnel"]; got < 1 {
		t.Fatalf("expected labor_personnel count >= 1 from seeded title, got %d", got)
	}
	if got := countByLawType["criminal_case"]; got != 0 {
		t.Fatalf("expected criminal_case count = 0 (no seeded match), got %d", got)
	}
}

func TestGetParsedLawReturnsNotFoundWhenVersionIDDoesNotExist(t *testing.T) {
	router := newTestRouter(t, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/laws/not-exists/parsed", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status code: got %d want %d", recorder.Code, http.StatusNotFound)
	}
}

func newTestRouter(t *testing.T, detailDir string) *gin.Engine {
	t.Helper()

	dbPath := createTestSQLiteDatabase(t)
	db, err := database.OpenReadOnlySQLite(dbPath)
	if err != nil {
		t.Fatalf("open read-only sqlite: %v", err)
	}

	typeRepo := repository.NewTypeRepository(db)
	lawRepo := repository.NewLawRepository(db)
	parsedLawRepo := repository.NewParsedLawRepository(detailDir)
	lawService := service.NewLawService(typeRepo, lawRepo, parsedLawRepo)
	lawHandler := handler.NewLawHandler(lawService)

	return server.NewRouter(lawHandler)
}

func createTestSQLiteDatabase(t *testing.T) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "laws.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test sqlite: %v", err)
	}

	for _, statement := range []string{
		`CREATE TABLE laws_list (
			versionId TEXT PRIMARY KEY,
			title TEXT,
			lawTypeId INTEGER,
			lawType TEXT,
			publishDate TEXT,
			effectDate TEXT,
			detailJson TEXT,
			effectiveStatus INTEGER,
			authorityId INTEGER,
			authorityName TEXT,
			parse_state INTEGER DEFAULT 0
		);`,
		`CREATE TABLE types (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id INTEGER
		);`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("execute schema statement: %v", err)
		}
	}

	insertType(t, db, 100, "宪法", nil)
	insertType(t, db, 101, "法律(全选)", nil)
	insertType(t, db, 102, "法律", intPtr(101))
	insertType(t, db, 110, "宪法相关法", intPtr(102))
	insertType(t, db, 120, "民法商法", intPtr(102))
	insertType(t, db, 130, "行政法", intPtr(102))
	insertType(t, db, 140, "经济法", intPtr(102))
	insertType(t, db, 150, "社会法", intPtr(102))
	insertType(t, db, 160, "刑法", intPtr(102))
	insertType(t, db, 170, "诉讼与非诉讼法", intPtr(102))
	insertType(t, db, 222, "地方法规", nil)
	insertType(t, db, 230, "地方性法规", intPtr(222))

	insertLaw(t, db, "law-parsed-1", "解析文件法律", 110, "法律", "2024-01-02", "2024-01-03", 1, "全国人大")
	insertLaw(t, db, "law-parsed-missing", "缺失解析文件法律", 110, "法律", "2024-01-01", "2024-01-02", 1, "全国人大")
	insertLaw(t, db, "law-110-1", "宪法相关法一", 110, "法律", "2024-01-05", "2024-01-06", 1, "全国人大")
	insertLaw(t, db, "law-110-2", "宪法相关法二", 110, "法律", "2024-01-04", "2024-01-05", 1, "全国人大")

	insertLaw(t, db, "law-120-a", "民法商法 A", 120, "法律", "2024-03-01", "", 1, "国务院")
	insertLaw(t, db, "law-120-b", "民法商法 B", 120, "法律", "2024-01-01", "2024-02-01", 1, "国务院")
	insertLaw(t, db, "law-120-c", "民法商法 C", 120, "法律", "2024-04-01", "", 1, "国务院")

	for i := 1; i <= 22; i++ {
		versionID := fmt.Sprintf("law-230-%02d", i)
		title := fmt.Sprintf("地方性法规 %02d", i)
		date := fmt.Sprintf("2024-03-%02d", i)
		insertLaw(t, db, versionID, title, 230, "地方性法规", date, date, 1, "地方人大")
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	future := now.AddDate(0, 1, 0).Format("2006-01-02")
	insertLaw(t, db, "law-new-recent", "最新发布法律", 110, "法律", today, future, 1, "全国人大")

	// 给「常用法律」的关键字统计准备命中数据；挂在 110 上避免影响其它分类的排序/计数断言
	insertLaw(t, db, "law-marriage-1", "中华人民共和国婚姻法", 110, "法律", "2001-04-28", "2001-04-28", 1, "全国人大")
	insertLaw(t, db, "law-labor-1", "中华人民共和国劳动合同法", 110, "法律", "2007-06-29", "2008-01-01", 1, "全国人大")

	return dbPath
}

func insertType(t *testing.T, db *gorm.DB, id int, name string, parentID *int) {
	t.Helper()

	if err := db.Exec(
		`INSERT INTO types(id, name, parent_id) VALUES (?, ?, ?)`,
		id, name, parentID,
	).Error; err != nil {
		t.Fatalf("insert type %d: %v", id, err)
	}
}

func insertLaw(t *testing.T, db *gorm.DB, versionID, title string, lawTypeID int, lawType, publishDate, effectDate string, effectiveStatus int, authorityName string) {
	t.Helper()

	if err := db.Exec(
		`INSERT INTO laws_list(versionId, title, lawTypeId, lawType, publishDate, effectDate, detailJson, effectiveStatus, authorityId, authorityName, parse_state)
		 VALUES (?, ?, ?, ?, ?, ?, '', ?, 0, ?, 1)`,
		versionID, title, lawTypeID, lawType, publishDate, effectDate, effectiveStatus, authorityName,
	).Error; err != nil {
		t.Fatalf("insert law %s: %v", versionID, err)
	}
}

func writeTypeJSONFile(t *testing.T, dir string, lawTypeID int, versionID string, data []byte) {
	t.Helper()

	filePath := filepath.Join(dir, "laws_by_type", fmt.Sprintf("type_%d", lawTypeID), versionID+".json")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create parsed law dir: %v", err)
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("write parsed law json: %v", err)
	}
}

func writeFlatJSONFile(t *testing.T, dir, versionID string, data []byte) {
	t.Helper()

	filePath := filepath.Join(dir, versionID+".json")
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("write parsed law json: %v", err)
	}
}

func decodeJSON(t *testing.T, data []byte, target any) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func assertIntSlice(t *testing.T, got, want []int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected slice length: got %d want %d", len(got), len(want))
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected slice value at %d: got %d want %d", i, got[i], want[i])
		}
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected slice length: got %d want %d", len(got), len(want))
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("unexpected slice value at %d: got %s want %s", i, got[i], want[i])
		}
	}
}

func intPtr(v int) *int {
	return &v
}
