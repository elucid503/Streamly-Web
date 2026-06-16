package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"streamly/internal/captions"
	"streamly/internal/config"
	"streamly/internal/database"
	"streamly/internal/handlers"
	"streamly/internal/middleware"
	"streamly/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	cfg, err := config.Load()

	if err != nil {

		log.Fatal(err)

	}

	if cfg.MongoURI == "" {

		log.Fatal("MONGO_URI is required")

	}

	db, err := database.Connect(cfg.MongoURI)

	if err != nil {

		log.Fatal(err)

	}

	authSvc := services.NewAuthService(db, cfg)
	settingsSvc := services.NewSettingsService(db)
	historySvc := services.NewHistoryService(db)
	favoritesSvc := services.NewFavoritesService(db)
	mediaSvc := services.NewMediaService(cfg)

	cacheCtx, cacheCancel := context.WithCancel(context.Background())

	defer cacheCancel()

	mediaSvc.StartCatalogCache(cacheCtx)
	defer mediaSvc.StopCatalogCache()

	proxySvc := services.NewProxyService(cfg)
	subdlClient := captions.NewSubDLClient(captions.SubDLOptions{APIKey: cfg.SubDLAPIKey})
	subtitleSvc := services.NewSubtitleResolver(mediaSvc, subdlClient, cfg)

	authHandler := handlers.NewAuthHandler(authSvc)
	settingsHandler := handlers.NewSettingsHandler(settingsSvc)
	historyHandler := handlers.NewHistoryHandler(historySvc)
	favoritesHandler := handlers.NewFavoritesHandler(favoritesSvc)
	catalogHandler := handlers.NewCatalogHandler(mediaSvc)
	streamHandler := handlers.NewStreamHandler(mediaSvc, proxySvc, subtitleSvc)
	proxyHandler := handlers.NewProxyHandler(proxySvc)
	adminHandler := handlers.NewAdminHandler(authSvc)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	r.Use(cors.New(cors.Config{

		AllowOrigins: []string{cfg.FrontendOrigin},

		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization", "Range", "If-Range"},
		ExposeHeaders: []string{"Content-Length", "Content-Range", "Accept-Ranges"},

		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	versionBytes, _ := os.ReadFile("version.txt")
	version := strings.TrimSpace(string(versionBytes))

	if version == "" {

		version = "1.0.0"

	}

	// Health

	r.GET("/health", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{"status": "ok"})

	})

	// API Bases

	api := r.Group("/api")

	api.GET("/version", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{"version": version})

	})
	auth := api.Group("/auth")

	// Auth

	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/me", middleware.AuthRequired(authSvc), authHandler.Me)

	// Protected Routes

	protected := api.Group("")
	protected.Use(middleware.AuthRequired(authSvc))

	protected.GET("/settings", settingsHandler.Get)
	protected.PUT("/settings", settingsHandler.Update)

	protected.GET("/history", historyHandler.List)
	protected.POST("/history", historyHandler.Upsert)
	protected.DELETE("/history/:id", historyHandler.Delete)

	protected.GET("/favorites", favoritesHandler.List)
	protected.POST("/favorites", favoritesHandler.Upsert)
	protected.DELETE("/favorites/:kind/:key", favoritesHandler.Delete)

	// Catalog

	protected.GET("/search", catalogHandler.Search)

	movies := protected.Group("/movies")

	movies.GET("/trending", catalogHandler.MovieTrending)
	movies.GET("/categories", catalogHandler.MovieCategories)
	movies.GET("/categories/:id", catalogHandler.MovieCategoryTitles)
	movies.GET("/:id", catalogHandler.MovieDetails)
	movies.GET("/:id/stream", streamHandler.MovieStream)
	movies.GET("/:id/subtitles", streamHandler.MovieSubtitles)
	movies.GET("/:id/intro", streamHandler.MovieIntro)

	shows := protected.Group("/shows")

	shows.GET("/trending", catalogHandler.ShowTrending)
	shows.GET("/categories", catalogHandler.ShowCategories)
	shows.GET("/categories/:id", catalogHandler.ShowCategoryTitles)
	shows.GET("/:id", catalogHandler.ShowDetails)
	shows.GET("/:id/seasons", catalogHandler.ShowSeasons)
	shows.GET("/:id/seasons/:season/episodes", catalogHandler.SeasonEpisodes)
	shows.GET("/:id/seasons/:season/episodes/:episode", catalogHandler.EpisodeDetails)
	shows.GET("/:id/seasons/:season/episodes/:episode/stream", streamHandler.EpisodeStream)
	shows.GET("/:id/seasons/:season/episodes/:episode/subtitles", streamHandler.EpisodeSubtitles)
	shows.GET("/:id/seasons/:season/episodes/:episode/intro", streamHandler.EpisodeIntro)
	shows.GET("/:id/seasons/:season/episodes/:episode/next", streamHandler.NextEpisode)

	// Live Channels

	live := protected.Group("/live")

	live.GET("/channels", catalogHandler.LiveChannels)
	live.GET("/channels/popular", catalogHandler.LivePopular)
	live.GET("/channels/search", catalogHandler.LiveSearch)
	live.GET("/channels/:id/stream", streamHandler.LiveStream)

	// Admin

	admin := protected.Group("/admin")
	admin.Use(middleware.AdminRequired())

	admin.POST("/access-codes", adminHandler.CreateAccessCode)
	admin.GET("/access-codes", adminHandler.ListAccessCodes)
	admin.DELETE("/access-codes/:code", adminHandler.DeleteAccessCode)

	// Proxy

	api.GET("/proxy/:token", proxyHandler.Serve)

	// SPA

	staticDir := cfg.StaticDir

	r.NoRoute(func(c *gin.Context) {

		if strings.HasPrefix(c.Request.URL.Path, "/api") {

			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return

		}

		filePath := filepath.Join(staticDir, filepath.Clean("/"+c.Request.URL.Path))

		info, err := os.Stat(filePath)

		if err == nil && !info.IsDir() {

			c.File(filePath)
			return

		}

		c.File(filepath.Join(staticDir, "index.html"))

	})

	server := &http.Server{

		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {

		log.Printf("streamly backend listening on :%s", cfg.Port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {

			log.Fatal(err)

		}

	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	_ = db.Close(ctx)

}
