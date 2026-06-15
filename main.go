package main

import (
	"context"
	"errors"
	"log"
	"os"

	"LawHelperServer/internal/config"
	"LawHelperServer/internal/database"
	"LawHelperServer/internal/handler"
	"LawHelperServer/internal/repository"
	"LawHelperServer/internal/server"
	"LawHelperServer/internal/service"
)

func main() {
	cfg := config.Load()
	logParsedLawDirStatus(cfg.LawDetailJSONDir)

	// 使用可创建表的数据库连接（支持读写和表创建）
	db, err := database.OpenSQLiteCreate(cfg.LawDBPath)
	if err != nil {
		log.Fatalf("open sqlite database: %v", err)
	}

	typeRepo := repository.NewTypeRepository(db)
	lawRepo := repository.NewLawRepository(db)
	parsedLawRepo := repository.NewParsedLawRepository(cfg.LawDetailJSONDir)
	commonLawRepo := repository.NewCommonLawRepository(db)
	localAuthorityRepo := repository.NewLocalAuthorityRepository(db)

	// 启动时同步常用法律数据和地方法律机关数据
	syncService := service.NewSyncService(commonLawRepo, lawRepo, localAuthorityRepo)
	if err := syncService.SyncCommonLaws(context.Background()); err != nil {
		log.Printf("Warning: sync common laws failed: %v", err)
	}
	if err := syncService.SyncLocalAuthorities(context.Background()); err != nil {
		log.Printf("Warning: sync local authorities failed: %v", err)
	}

	lawService := service.NewLawService(typeRepo, lawRepo, parsedLawRepo, commonLawRepo, localAuthorityRepo)
	lawHandler := handler.NewLawHandler(lawService)
	router := server.NewRouter(lawHandler)

	log.Printf("parsed law json dir: %s", cfg.LawDetailJSONDir)
	log.Printf("law server listening on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("run http server: %v", err)
	}
}

func logParsedLawDirStatus(dir string) {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			log.Printf("parsed law json path exists but is not a directory: %s", dir)
		}
		return
	}

	if errors.Is(err, os.ErrNotExist) {
		log.Printf("parsed law json dir does not exist yet: %s", dir)
		log.Printf("place parsed files as <versionId>.json under that directory")
		return
	}

	log.Printf("check parsed law json dir %s: %v", dir, err)
}
