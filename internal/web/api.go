package web

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) registerAPI() {
	api := s.engine.Group("/api")
	api.GET("/stats", s.handleGetStats)
	api.GET("/repos", s.handleGetRepos)
	api.POST("/sync", s.handleSync)
}

func (s *Server) handleGetStats(c *gin.Context) {
	repo := c.Query("repo")
	if repo == "" {
		c.JSON(http.StatusOK, gin.H{"stats": s.stats})
		return
	}
	for _, st := range s.stats {
		if st.Repo == repo {
			c.JSON(http.StatusOK, st)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
}

func (s *Server) handleGetRepos(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"repos": s.repos})
}

func (s *Server) handleSync(c *gin.Context) {
	repo := c.Query("repo")
	go func() {
		if repo != "" {
			_ = s.syncer.SyncRepo(context.Background(), repo)
		} else {
			s.syncer.SyncAll(context.Background(), s.repos, nil, 3)
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started"})
}
