package controllers

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"ships3d/config"
	dataaccess "ships3d/dataAccess"
	"ships3d/game"
	"ships3d/i18n"
)

// maxBodyBytes caps a request body. The largest legitimate one is a project
// at its part limit, which is well under this.
const maxBodyBytes = 1 << 20

type Server struct {
	cfg   config.Config
	store *dataaccess.Store
	hub   *game.Hub
	arena *game.Arena
}

func NewServer(cfg config.Config, store *dataaccess.Store) *Server {
	s := &Server{cfg: cfg, store: store}
	s.hub = game.NewHub(game.DefaultSettings(), s.loadShip, s.identify, s.checkOrigin)
	s.arena = game.NewArena(game.DefaultSettings(), game.DefaultAssaultRules(), s.loadShip, s.identify, s.checkOrigin)
	return s
}

func (s *Server) Hub() *game.Hub { return s.hub }

func (s *Server) Arena() *game.Arena { return s.arena }

func (s *Server) Router() *gin.Engine {
	router := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = s.cfg.AllowedOrigins
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("GET", "POST", "PUT", "DELETE", "OPTIONS")
	corsConfig.AddAllowHeaders("Accept-Language")
	if len(corsConfig.AllowOrigins) == 0 {
		// gin-contrib/cors refuses an empty list; with no configured
		// origins, only same-origin requests (which CORS doesn't affect)
		// can use the API.
		corsConfig.AllowOriginFunc = func(string) bool { return false }
	}
	router.Use(cors.New(corsConfig))
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	})

	router.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "players": s.hub.PlayerCount(), "assaultPlayers": s.arena.PlayerCount()})
	})
	s.registerUserRoutes(router)
	s.registerProjectRoutes(router)
	s.registerGameRoutes(router)
	return router
}

// checkOrigin decides which pages may open a game websocket. Browsers always
// send Origin on a websocket handshake; a request without one is not from a
// browser page (e.g. a native client) and carries no ambient cookie risk.
func (s *Server) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if slices.Contains(s.cfg.AllowedOrigins, origin) {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// errorResponse answers with the messages in the language the request asks
// for (see i18n.FromRequest).
func errorResponse(c *gin.Context, status int, messages ...i18n.Message) {
	lang := i18n.FromRequest(c.Request)
	texts := make([]string, len(messages))
	for i, message := range messages {
		texts[i] = message.In(lang)
	}
	c.AbortWithStatusJSON(status, gin.H{"success": false, "errors": texts})
}
