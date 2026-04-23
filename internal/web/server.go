package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/cache"
	isync "github.com/jimyag/commitlens/internal/sync"
)

type Server struct {
	engine     *gin.Engine
	syncer     *isync.Syncer
	stats      []*cache.StatsData
	repos      []string
	frontendFS http.FileSystem
}

func New(assets embed.FS, syncer *isync.Syncer, stats []*cache.StatsData, repos []string) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		engine: gin.New(),
		syncer: syncer,
		stats:  stats,
		repos:  repos,
	}
	s.mountFrontend(assets)
	s.registerAPI()
	return s
}

func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}

func (s *Server) mountFrontend(assets embed.FS) {
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return
	}
	s.frontendFS = http.FS(sub)

	s.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := s.frontendFS.Open(path)
		if err != nil {
			// SPA fallback
			f, err = s.frontendFS.Open("index.html")
			if err != nil {
				http.NotFound(c.Writer, c.Request)
				return
			}
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			http.NotFound(c.Writer, c.Request)
			return
		}
		http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
	})
}
