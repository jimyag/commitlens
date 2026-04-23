package stats

import (
	"regexp"
	"strings"
	"unicode"

	gh "github.com/jimyag/commitlens/internal/github"
)

// coAuthoredByLine 匹配 "Co-authored-by: ... <...>" 行，大小写不敏感，忽略每行首尾的空白
var coAuthoredByLine = regexp.MustCompile(`(?i)^[ \t]*Co-authored-by:.*<([^>]+)>`)

// uniquePRParticipants 返回该 PR 的参与者 GitHub 登录名：主作者 + 从 Co-authored-by 解析出的
// 合著者。同一用户在同一条目或多次 Co-authored-by 中只出现一次。顺序：先作者，再合著者
// 按出现先后。
func uniquePRParticipants(pr *gh.PR) []string {
	seen := make(map[string]struct{}, 8+len(pr.Commits))
	var out []string
	add := func(login string) {
		login = strings.TrimSpace(login)
		if login == "" {
			return
		}
		if _, ok := seen[login]; ok {
			return
		}
		seen[login] = struct{}{}
		out = append(out, login)
	}
	add(pr.Author)
	for _, login := range coauthorLoginsFromPR(pr) {
		add(login)
	}
	return out
}

// coauthorLoginsFromPR 从该 PR 下所有 commit message 的 Co-authored-by 中解析出 GitHub
// 登录名；仅信任 users.noreply.github.com 形式（含 id+login），其它邮箱无法唯一定位 login 时忽略。
// 多行/多 commit 中重复的用户在返回前去重，保留先出现者顺序。
func coauthorLoginsFromPR(pr *gh.PR) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, c := range pr.Commits {
		for _, line := range strings.Split(c.Message, "\n") {
			line = strings.TrimRightFunc(line, unicode.IsSpace)
			login, ok := loginFromCoAuthoredTrailerLine(line)
			if !ok {
				continue
			}
			if _, d := seen[login]; d {
				continue
			}
			seen[login] = struct{}{}
			out = append(out, login)
		}
	}
	return out
}

func loginFromCoAuthoredTrailerLine(line string) (string, bool) {
	line = strings.TrimRightFunc(line, unicode.IsSpace)
	if line == "" {
		return "", false
	}
	m := coAuthoredByLine.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	email := strings.TrimSpace(m[1])
	// 只要常见 GitHub 伪装邮箱
	login, ok := loginFromNoreplyEmail(email)
	if !ok {
		return "", false
	}
	return login, true
}

// loginFromNoreplyEmail 从 GitHub noreply 邮箱中取出 login（与 Git 提交里 Co-authored-by 常见格式一致）
func loginFromNoreplyEmail(email string) (string, bool) {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 {
		return "", false
	}
	local, domain := email[:at], email[at+1:]
	if !strings.EqualFold(domain, "users.noreply.github.com") {
		return "", false
	}
	// 形式：{数字id}+username@ 或 直接 username@（无 +）
	if i := strings.IndexByte(local, '+'); i > 0 {
		if isAllDigitASCII(local[:i]) {
			rest := local[i+1:]
			if rest == "" {
				return "", false
			}
			return rest, true
		}
	}
	if local == "" {
		return "", false
	}
	return local, true
}

func isAllDigitASCII(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
