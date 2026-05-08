package web

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/commitlens/internal/stats"
)

func (s *Server) registerAPI() {
	api := s.engine.Group("/api")
	api.GET("/stats", s.handleGetStats)
	api.GET("/repos", s.handleGetRepos)
	api.GET("/prs", s.handleGetPRs)
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
			s.syncer.SyncAll(context.Background(), s.repos, nil, 5)
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"message": "sync started"})
}

// PRInfo 是对外暴露的 PR 摘要，不含 commits 代码细节。
type PRInfo struct {
	Repo         string    `json:"repo"`
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	AvatarURL    string    `json:"avatar_url"`
	Participants []string  `json:"participants"`
	MergedAt     time.Time `json:"merged_at"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
}

func (s *Server) handleGetPRs(c *gin.Context) {
	repo := c.Query("repo")   // 可选；空 = 全部仓库
	login := c.Query("login") // 可选；空 = 不按贡献者过滤
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var hasFrom, hasTo bool
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from, expect RFC3339"})
			return
		}
		from, hasFrom = t, true
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to, expect RFC3339"})
			return
		}
		to, hasTo = t, true
	}

	repos := s.repos
	if repo != "" {
		repos = []string{repo}
	}

	var result []PRInfo
	for _, r := range repos {
		raw, err := s.rawCache.Load(r)
		if err != nil {
			continue
		}
		for _, pr := range raw.PRs {
			if hasFrom && pr.MergedAt.Before(from) {
				continue
			}
			if hasTo && !pr.MergedAt.Before(to) {
				continue
			}
			if login != "" {
				found := false
				for _, p := range stats.PRParticipants(&pr) {
					if p == login {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			result = append(result, PRInfo{
				Repo:         r,
				Number:       pr.Number,
				Title:        pr.Title,
				Author:       pr.Author,
				AvatarURL:    pr.AvatarURL,
				Participants: stats.PRParticipants(&pr),
				MergedAt:     pr.MergedAt,
				Additions:    pr.Additions,
				Deletions:    pr.Deletions,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].MergedAt.After(result[j].MergedAt)
	})

	total := len(result)

	page := 1
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && p > 0 {
		page = p
	}
	perPage := 100
	if pp, err := strconv.Atoi(c.DefaultQuery("per_page", "100")); err == nil && pp > 0 && pp <= 100 {
		perPage = pp
	}

	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, gin.H{
		"prs":      result[start:end],
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
