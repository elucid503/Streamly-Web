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

	"streamly/internal/config"
	"streamly/internal/database"
	"streamly/internal/features/admin"
	"streamly/internal/features/auth"
	"streamly/internal/features/catalog"
	discover "streamly/internal/features/catalog/discovery"
	"streamly/internal/features/library"
	"streamly/internal/features/playback"
	"streamly/internal/features/playback/captions"
	"streamly/internal/features/settings"
	"streamly/internal/features/sports"
	"streamly/internal/middleware"

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

	authSvc := auth.NewAuthService(db, cfg)
	settingsSvc := settings.NewSettingsService(db)
	historySvc := library.NewHistoryService(db)
	favoritesSvc := library.NewFavoritesService(db)
	mediaSvc := catalog.NewMediaService(cfg)

	cacheCtx, cacheCancel := context.WithCancel(context.Background())

	defer cacheCancel()

	mediaSvc.StartCatalogCache(cacheCtx)
	defer mediaSvc.StopCatalogCache()

	proxySvc := playback.NewProxyService(cfg)
	subdlClient := captions.NewSubDLClient(captions.SubDLOptions{APIKey: cfg.SubDLAPIKey})
	opensubsClient := captions.NewOpenSubsClient(captions.OpenSubsOptions{APIKey: cfg.OpenSubtitlesAPIKey})

	if !subdlClient.Configured() && !opensubsClient.Configured() {

		log.Println("warning: no subtitle API keys configured (SUBDL_API_KEY / OPENSUBTITLES_API_KEY); subtitles disabled")

	} else if subdlClient.Configured() {

		log.Println("subtitles: SubDL configured")

	} else {

		log.Println("subtitles: OpenSubtitles configured")

	}

	subtitleSvc := playback.NewSubtitleResolver(mediaSvc, subdlClient, opensubsClient, cfg)

	if strings.EqualFold(strings.TrimSpace(os.Getenv("STREAM_DEBUG")), "1") || strings.EqualFold(strings.TrimSpace(os.Getenv("STREAM_DEBUG")), "true") {

		log.Println("stream: debug logging enabled (STREAM_DEBUG=1)")

	}

	authHandler := auth.NewAuthHandler(authSvc)
	settingsHandler := settings.NewSettingsHandler(settingsSvc)
	historyHandler := library.NewHistoryHandler(historySvc)
	favoritesHandler := library.NewFavoritesHandler(favoritesSvc)
	catalogHandler := catalog.NewCatalogHandler(mediaSvc)
	streamHandler := playback.NewStreamHandler(mediaSvc, proxySvc, settingsSvc, subtitleSvc)
	proxyHandler := playback.NewProxyHandler(proxySvc)
	adminHandler := admin.NewAdminHandler(authSvc, db)

	feedSvc := discover.New(mediaSvc.Client(), filepath.Join("data", "bridge.cache.json"), mediaSvc)
	feedSvc.Start(cacheCtx, cfg.CatalogCacheTTL)
	defer feedSvc.Stop()

	feedHandler := catalog.NewFeedHandler(feedSvc)

	pushSvc := sports.NewPushService(db, cfg)

	if !pushSvc.Configured() {

		log.Println("warning: VAPID_PUBLIC_KEY / VAPID_PRIVATE_KEY unset; sports kickoff alerts disabled")

	}

	sportsAlertsSvc := sports.NewSportsAlertsService(db, pushSvc, mediaSvc)
	sportsAlertsSvc.Start(cacheCtx)
	defer sportsAlertsSvc.Stop()

	sportsAlertsHandler := sports.NewSportsAlertsHandler(sportsAlertsSvc, pushSvc)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	r.Use(cors.New(cors.Config{

		AllowOrigins: []string{cfg.FrontendOrigin},

		AllowMethods: []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "Range", "If-Range"},
		ExposeHeaders: []string{"Content-Length", "Content-Range", "Accept-Ranges"},

		AllowCredentials: true,
		MaxAge: 12 * time.Hour,

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
	authRoutes := api.Group("/auth")

	// Auth

	authRoutes.POST("/register", middleware.AuthRateLimit, authHandler.Register)
	authRoutes.POST("/login", middleware.AuthRateLimit, authHandler.Login)
	authRoutes.POST("/logout", authHandler.Logout)
	authRoutes.GET("/me", auth.AuthRequired(authSvc), authHandler.Me)

	// Protected Routes

	protected := api.Group("")
	protected.Use(auth.AuthRequired(authSvc))

	protected.GET("/settings", settingsHandler.Get)
	protected.PUT("/settings", settingsHandler.Update)

	protected.GET("/history", historyHandler.List)
	protected.POST("/history", historyHandler.Upsert)
	protected.DELETE("/history/:id", historyHandler.Delete)

	protected.GET("/favorites", favoritesHandler.List)
	protected.POST("/favorites", favoritesHandler.Upsert)
	protected.DELETE("/favorites/:kind/:key", favoritesHandler.Delete)

	// Catalog

	protected.GET("/search", middleware.SearchRateLimit, catalogHandler.Search)

	protected.GET("/feed/movies", feedHandler.Movies)
	protected.GET("/feed/shows", feedHandler.Shows)
	protected.GET("/resolve", middleware.SearchRateLimit, feedHandler.Resolve)

	movies := protected.Group("/movies")

	movies.GET("/trending", catalogHandler.MovieTrending)
	movies.GET("/categories", catalogHandler.MovieCategories)
	movies.GET("/categories/:id", catalogHandler.MovieCategoryTitles)
	movies.GET("/:id", catalogHandler.MovieDetails)
	movies.GET("/:id/stream", middleware.StreamRateLimit, streamHandler.MovieStream)
	movies.GET("/:id/subtitles", middleware.StreamRateLimit, streamHandler.MovieSubtitles)
	movies.GET("/:id/intro", middleware.StreamRateLimit, streamHandler.MovieIntro)

	shows := protected.Group("/shows")

	shows.GET("/trending", catalogHandler.ShowTrending)
	shows.GET("/categories", catalogHandler.ShowCategories)
	shows.GET("/categories/:id", catalogHandler.ShowCategoryTitles)
	shows.GET("/:id", catalogHandler.ShowDetails)
	shows.GET("/:id/seasons", catalogHandler.ShowSeasons)
	shows.GET("/:id/seasons/:season/episodes", catalogHandler.SeasonEpisodes)
	shows.GET("/:id/seasons/:season/episodes/:episode", catalogHandler.EpisodeDetails)
	shows.GET("/:id/seasons/:season/episodes/:episode/stream", middleware.StreamRateLimit, streamHandler.EpisodeStream)
	shows.GET("/:id/seasons/:season/episodes/:episode/subtitles", middleware.StreamRateLimit, streamHandler.EpisodeSubtitles)
	shows.GET("/:id/seasons/:season/episodes/:episode/intro", middleware.StreamRateLimit, streamHandler.EpisodeIntro)
	shows.GET("/:id/seasons/:season/episodes/:episode/next", streamHandler.NextEpisode)

	// Live Channels

	live := protected.Group("/live")

	live.GET("/channels", catalogHandler.LiveChannels)
	live.GET("/channels/popular", catalogHandler.LivePopular)
	live.GET("/channels/search", middleware.SearchRateLimit, catalogHandler.LiveSearch)
	live.GET("/channels/:id/stream", middleware.StreamRateLimit, streamHandler.LiveStream)
	live.GET("/providers", streamHandler.LiveProviders)
	live.GET("/schedule", catalogHandler.LiveSchedule)
	live.GET("/sports", catalogHandler.LiveSports)

	protected.GET("/push/vapid", sportsAlertsHandler.VapidPublicKey)
	protected.PUT("/push/subscription", sportsAlertsHandler.UpsertSubscription)
	protected.DELETE("/push/subscription", sportsAlertsHandler.DeleteSubscription)

	protected.GET("/sports/alerts", sportsAlertsHandler.List)
	protected.PUT("/sports/alerts/teams", sportsAlertsHandler.SubscribeTeam)
	protected.DELETE("/sports/alerts/teams", sportsAlertsHandler.UnsubscribeTeam)
	protected.PUT("/sports/alerts/:matchId", sportsAlertsHandler.Subscribe)
	protected.DELETE("/sports/alerts/:matchId", sportsAlertsHandler.Unsubscribe)

	// Admin

	admin := protected.Group("/admin")
	admin.Use(auth.AdminRequired())

	admin.POST("/access-codes", adminHandler.CreateAccessCode)
	admin.GET("/access-codes", adminHandler.ListAccessCodes)
	admin.DELETE("/access-codes/:code", adminHandler.DeleteAccessCode)

	admin.GET("/service-interruption", adminHandler.GetServiceInterruption)
	admin.PUT("/service-interruption", adminHandler.UpdateServiceInterruption)

	protected.GET("/service-interruption", adminHandler.GetServiceInterruption)

	// Proxy

	api.GET("/proxy/:token", proxyHandler.Serve)
	api.HEAD("/proxy/:token", proxyHandler.Serve)

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

		Addr: ":" + cfg.Port,
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
