package main

import (
	"crypto/rand"
	"encoding/hex"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

type accountInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}
type managedUser struct {
	Reporter
	Disabled   bool `json:"disabled"`
	IssueCount int  `json:"issueCount"`
}

func validPassword(password string) bool {
	return utf8.RuneCountInString(password) >= 6 && len(password) <= 72
}
func (a *inspectorAuth) createAccount(in accountInput) (*Reporter, int, string) {
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if !regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`).MatchString(in.Username) {
		return nil, 400, "账号需为3～32位字母、数字或下划线，并以字母开头"
	}
	if in.Username == "admin" {
		return nil, 400, "该账号名称不可用于巡检员注册"
	}
	if utf8.RuneCountInString(in.DisplayName) < 1 || utf8.RuneCountInString(in.DisplayName) > 40 {
		return nil, 400, "姓名需为1～40个字符"
	}
	if !validPassword(in.Password) {
		return nil, 400, "密码至少6个字符，且不超过72字节"
	}
	for _, user := range a.users {
		if strings.EqualFold(user.Username, in.Username) {
			return nil, 409, "该账号已存在，请使用其他账号"
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, 500, "密码处理失败"
	}
	raw := make([]byte, 16)
	if _, err = rand.Read(raw); err != nil {
		return nil, 500, "无法创建用户编号"
	}
	user := inspectorAccount{Reporter: Reporter{"inspector-" + hex.EncodeToString(raw), in.Username, in.DisplayName}, PasswordHash: string(hash)}
	next := append(append([]inspectorAccount{}, a.users...), user)
	if err = a.saveUsers(next); err != nil {
		return nil, 500, "账号保存失败，请检查服务端后重试"
	}
	return &user.Reporter, 201, ""
}
func (a *inspectorAuth) revokeUser(id string) {
	for token, session := range a.sessions {
		if session.User.ID == id {
			delete(a.sessions, token)
		}
	}
}
func (s *Server) userSummary(user inspectorAccount) managedUser {
	count := 0
	for _, issue := range s.issues {
		if issue.Reporter != nil && issue.Reporter.ID == user.ID {
			count++
		}
	}
	return managedUser{user.Reporter, user.Disabled, count}
}
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	a := s.inspectors
	if a == nil {
		fail(w, 503, "巡检员账户服务尚未就绪")
		return
	}
	if r.URL.Path == "/api/admin/users" {
		switch r.Method {
		case "GET":
			list := []managedUser{}
			for _, user := range a.users {
				list = append(list, s.userSummary(user))
			}
			reply(w, 200, list)
		case "POST":
			var in accountInput
			if err := decode(w, r, &in); err != nil {
				fail(w, 400, "账号参数不正确")
				return
			}
			user, code, message := a.createAccount(in)
			if user == nil {
				fail(w, code, message)
				return
			}
			reply(w, 201, s.userSummary(a.users[len(a.users)-1]))
		default:
			fail(w, 405, "method not allowed")
		}
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), "/")
	if len(parts) > 2 || parts[0] == "" {
		fail(w, 404, "route not found")
		return
	}
	index := -1
	for i, user := range a.users {
		if user.ID == parts[0] {
			index = i
			break
		}
	}
	if index < 0 {
		fail(w, 404, "用户不存在")
		return
	}
	next := append([]inspectorAccount{}, a.users...)
	if len(parts) == 2 && parts[1] == "password" {
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return
		}
		var in struct {
			Password string `json:"password"`
		}
		if err := decode(w, r, &in); err != nil || !validPassword(in.Password) {
			fail(w, 400, "密码至少6个字符，且不超过72字节")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(w, 500, "密码处理失败")
			return
		}
		next[index].PasswordHash = string(hash)
	} else if len(parts) == 1 {
		if r.Method != "PATCH" {
			fail(w, 405, "method not allowed")
			return
		}
		var in struct {
			DisplayName *string `json:"displayName"`
			Disabled    *bool   `json:"disabled"`
		}
		if err := decode(w, r, &in); err != nil || (in.DisplayName == nil && in.Disabled == nil) {
			fail(w, 400, "请提供姓名或账户状态")
			return
		}
		if in.DisplayName != nil {
			name := strings.TrimSpace(*in.DisplayName)
			if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 40 {
				fail(w, 400, "姓名需为1～40个字符")
				return
			}
			next[index].DisplayName = name
		}
		if in.Disabled != nil {
			next[index].Disabled = *in.Disabled
		}
	} else {
		fail(w, 404, "route not found")
		return
	}
	if err := a.saveUsers(next); err != nil {
		fail(w, 500, "账号保存失败，修改未生效")
		return
	}
	a.revokeUser(parts[0])
	reply(w, 200, s.userSummary(a.users[index]))
}
