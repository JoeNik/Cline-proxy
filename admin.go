package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// In-memory OAuth login state for async browser login
var (
	oauthSessions   = make(map[string]*oauthSessionState)
	oauthSessionsMu sync.Mutex
)

type oauthSessionState struct {
	DeviceCode string
	UserCode   string
	AuthURL    string
	CreatedAt  time.Time
	Done       bool
	Success    bool
	Email      string
	Error      string
}

type apiResponse struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func writeAPI(w http.ResponseWriter, status int, resp apiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func registerAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin/", adminStaticHandler)
	mux.HandleFunc("/admin", adminStaticHandler)
	mux.HandleFunc("/admin/api/auth/verify", corsHandler(handleAuthVerify))
	mux.HandleFunc("/admin/api/auth/check", corsHandler(handleAuthCheck))
	mux.HandleFunc("/admin/api/accounts", corsHandler(authMiddleware(handleAdminAccounts)))
	mux.HandleFunc("/admin/api/accounts/add", corsHandler(authMiddleware(handleAdminAccountAdd)))
	mux.HandleFunc("/admin/api/accounts/delete", corsHandler(authMiddleware(handleAdminAccountDelete)))
	mux.HandleFunc("/admin/api/oauth/start", corsHandler(authMiddleware(handleOAuthStart)))
	mux.HandleFunc("/admin/api/oauth/status", corsHandler(authMiddleware(handleOAuthStatus)))
	mux.HandleFunc("/admin/api/sso/import", corsHandler(authMiddleware(handleSSOImport)))
	mux.HandleFunc("/admin/api/stats", corsHandler(authMiddleware(handleAdminStats)))
	mux.HandleFunc("/admin/api/batch-import", corsHandler(authMiddleware(handleBatchImport)))
	mux.HandleFunc("/admin/api/accounts/refresh-all", corsHandler(authMiddleware(handleAdminRefreshAll)))
	mux.HandleFunc("/admin/api/accounts/delete-all", corsHandler(authMiddleware(handleAdminDeleteAll)))
	mux.HandleFunc("/admin/api/accounts/reset", corsHandler(authMiddleware(handleAdminAccountReset)))
	mux.HandleFunc("/admin/api/keys", corsHandler(authMiddleware(handleAdminGetKeys)))
	mux.HandleFunc("/admin/api/keys/generate", corsHandler(authMiddleware(handleAdminGenerateKey)))
	mux.HandleFunc("/admin/api/keys/delete", corsHandler(authMiddleware(handleAdminDeleteKey)))
	mux.HandleFunc("/admin/api/models", corsHandler(authMiddleware(handleAdminModels)))
	mux.HandleFunc("/admin/api/models/official", corsHandler(authMiddleware(handleAdminModelsOfficial)))
	mux.HandleFunc("/admin/api/models/add", corsHandler(authMiddleware(handleAdminModelsAdd)))
	mux.HandleFunc("/admin/api/models/delete", corsHandler(authMiddleware(handleAdminModelsDelete)))
	mux.HandleFunc("/admin/api/config", corsHandler(authMiddleware(handleAdminConfig)))
	mux.HandleFunc("/admin/api/config/update", corsHandler(authMiddleware(handleAdminUpdateConfig)))
	mux.HandleFunc("/admin/api/scheduler/config", corsHandler(authMiddleware(handleSchedulerConfig)))
	mux.HandleFunc("/admin/api/scheduler/update", corsHandler(authMiddleware(handleSchedulerUpdate)))
	mux.HandleFunc("/admin/api/scheduler/trigger", corsHandler(authMiddleware(handleSchedulerTrigger)))
}

func adminStaticHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/" || r.URL.Path == "/admin" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(adminHTML))
		return
	}
	http.NotFound(w, r)
}

// GET /admin/api/accounts
func handleAdminAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	accounts := listAccounts()
	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Data: map[string]any{
			"accounts":   accounts,
			"total":      len(accounts),
			"poolIndex":  loadPool().CurrentIdx,
		},
	})
}

// POST /admin/api/accounts/add  body: { refreshToken, email }
func handleAdminAccountAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		RefreshToken string `json:"refreshToken"`
		Email        string `json:"email"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	if req.RefreshToken == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "refreshToken is required"})
		return
	}

	// Validate by refreshing
	resp, err := refreshClineToken(req.RefreshToken)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid refreshToken: " + err.Error()})
		return
	}

	if req.Email == "" {
		req.Email = fmt.Sprintf("user_%d", len(loadPool().Accounts)+1)
	}

	acc := &Account{
		AccountID:    fmt.Sprintf("acc_%d", time.Now().UnixMilli()),
		Email:        req.Email,
		RefreshToken: req.RefreshToken,
		AccessToken:  "workos:" + resp.Data.AccessToken,
		ExpiresAt:    parseExpiry(resp.Data.ExpiresAt) - 60000,
		Status:       "active",
		CreatedAt:    time.Now(),
	}
	if resp.Data.RefreshToken != "" {
		acc.RefreshToken = resp.Data.RefreshToken
	}

	addAccount(acc)
	log.Printf("Account added via API: %s", req.Email)

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: fmt.Sprintf("Account %s added", req.Email),
		Data: map[string]any{
			"accountId": acc.AccountID,
			"email":     acc.Email,
			"status":    acc.Status,
		},
	})
}

// POST /admin/api/accounts/delete  body: { accountId }
func handleAdminAccountDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	if req.AccountID == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "accountId is required"})
		return
	}

	if removeAccount(req.AccountID) {
		writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "Account deleted"})
	} else {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: "Account not found"})
	}
}

// POST /admin/api/oauth/start  -- Start OAuth device login, returns URL
func handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}

	device, err := workosDeviceAuth()
	if err != nil {
		writeAPI(w, http.StatusInternalServerError, apiResponse{Error: err.Error()})
		return
	}

	authURL := device.VerificationURIComplete
	if authURL == "" {
		authURL = device.VerificationURI
	}

	sessionID := fmt.Sprintf("oauth_%d", time.Now().UnixMilli())
	state := &oauthSessionState{
		DeviceCode: device.DeviceCode,
		UserCode:   device.UserCode,
		AuthURL:    authURL,
		CreatedAt:  time.Now(),
	}

	oauthSessionsMu.Lock()
	oauthSessions[sessionID] = state
	oauthSessionsMu.Unlock()

	// Start polling in background
	go func() {
		interval := device.Interval
		if interval < 5 {
			interval = 5
		}
		expiresIn := device.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 300
		}

		workosTok, err := pollWorkosToken(device.DeviceCode, interval, expiresIn)
		if err != nil {
			oauthSessionsMu.Lock()
			state.Error = err.Error()
			state.Done = true
			state.Success = false
			oauthSessionsMu.Unlock()
			return
		}

		cline, err := registerWithCline(workosTok.AccessToken, workosTok.RefreshToken)
		if err != nil {
			oauthSessionsMu.Lock()
			state.Error = err.Error()
			state.Done = true
			state.Success = false
			oauthSessionsMu.Unlock()
			return
		}

		email := "unknown"
		if cline.Data.UserInfo != nil && cline.Data.UserInfo.Email != "" {
			email = cline.Data.UserInfo.Email
		}

		acc := &Account{
			AccountID:    fmt.Sprintf("acc_%d", time.Now().UnixMilli()),
			Email:        email,
			RefreshToken: cline.Data.RefreshToken,
			AccessToken:  "workos:" + cline.Data.AccessToken,
			ExpiresAt:    parseExpiry(cline.Data.ExpiresAt) - 60000,
			Status:       "active",
			CreatedAt:    time.Now(),
		}
		addAccount(acc)

		oauthSessionsMu.Lock()
		state.Done = true
		state.Success = true
		state.Email = email
		oauthSessionsMu.Unlock()
		log.Printf("OAuth account added: %s", email)
	}()

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Data: map[string]any{
			"sessionId":       sessionID,
			"verificationUri": authURL,
			"userCode":        device.UserCode,
		},
	})
}

// GET /admin/api/oauth/status?sessionId=xxx
func handleOAuthStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "sessionId required"})
		return
	}

	oauthSessionsMu.Lock()
	state, ok := oauthSessions[sessionID]
	oauthSessionsMu.Unlock()

	if !ok {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: "session not found"})
		return
	}

	resp := map[string]any{
		"done":    state.Done,
		"success": state.Success,
	}
	if state.Done {
		resp["email"] = state.Email
		if !state.Success {
			resp["error"] = state.Error
		}
	}

	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: resp})
}

// POST /admin/api/sso/import  body: { ssoCookies: string, email?: string }
func handleSSOImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		SSOCookies string `json:"ssoCookies"`
		Email      string `json:"email"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	if req.SSOCookies == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "ssoCookies is required"})
		return
	}

	// SSO cookies import - try to use WorkOS device auth (requires browser)
	// For direct SSO cookie conversion, we'd need the WorkOS session cookie
	// to exchange for tokens. This is a placeholder that accepts WorkOS session
	// cookies. In practice, users should use OAuth or direct refreshToken.
	//
	// SSO cookie format expected: workos_session=xxx or similar
	lines := strings.Split(req.SSOCookies, "\n")
	imported := 0
	errors := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Try to use the cookie as a refresh token directly (common format)
		if strings.HasPrefix(line, "workos:") || len(line) > 20 {
			token := strings.TrimPrefix(line, "workos:")
			resp, err := refreshClineToken(token)
			if err != nil {
				errors = append(errors, fmt.Sprintf("token %s...: %v", truncate(token, 16), err))
				continue
			}
			email := req.Email
			if email == "" {
				email = fmt.Sprintf("sso_user_%d", time.Now().UnixMilli())
			}

			acc := &Account{
				AccountID:    fmt.Sprintf("acc_%d", time.Now().UnixMilli()),
				Email:        email,
				RefreshToken: token,
				AccessToken:  "workos:" + resp.Data.AccessToken,
				ExpiresAt:    parseExpiry(resp.Data.ExpiresAt) - 60000,
				Status:       "active",
				CreatedAt:    time.Now(),
			}
			addAccount(acc)
			imported++
		}
	}

	result := map[string]any{
		"imported": imported,
		"failed":   len(errors),
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: fmt.Sprintf("Imported %d accounts, %d failed", imported, len(errors)),
		Data:    result,
	})
}

// POST /admin/api/batch-import  body: { tokens: [{ refreshToken, email }] }
func handleBatchImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		Tokens []struct {
			RefreshToken string `json:"refreshToken"`
			Email        string `json:"email"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	if len(req.Tokens) == 0 {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "tokens array is empty"})
		return
	}

	imported := 0
	errors := []string{}

	for _, t := range req.Tokens {
		if t.RefreshToken == "" {
			continue
		}
		resp, err := refreshClineToken(t.RefreshToken)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", t.Email, err))
			continue
		}
		email := t.Email
		if email == "" {
			email = fmt.Sprintf("batch_%d", time.Now().UnixMilli())
		}
		acc := &Account{
			AccountID:    fmt.Sprintf("acc_%d", time.Now().UnixMilli()),
			Email:        email,
			RefreshToken: t.RefreshToken,
			AccessToken:  "workos:" + resp.Data.AccessToken,
			ExpiresAt:    parseExpiry(resp.Data.ExpiresAt) - 60000,
			Status:       "active",
			CreatedAt:    time.Now(),
		}
		addAccount(acc)
		imported++
	}

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: fmt.Sprintf("Imported %d accounts, %d failed", imported, len(errors)),
		Data: map[string]any{
			"imported": imported,
			"failed":   len(errors),
			"errors":   errors,
		},
	})
}

// POST /admin/api/accounts/refresh-all
func handleAdminRefreshAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	p := loadPool()
	poolMu.Lock()
	for _, a := range p.Accounts {
		if err := refreshAccountToken(a); err != nil {
			log.Printf("Refresh failed for %s: %v", a.Email, err)
		}
	}
	poolMu.Unlock()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "All tokens refreshed"})
}

// POST /admin/api/accounts/delete-all
func handleAdminDeleteAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	poolMu.Lock()
	pool = &AccountPool{Accounts: []*Account{}, Keys: []string{}}
	poolMu.Unlock()
	savePool()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "All accounts deleted"})
}

// POST /admin/api/accounts/reset  body: { accountId }
func handleAdminAccountReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	acc := getAccountByID(req.AccountID)
	if acc == nil {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: "account not found"})
		return
	}

	// Reset status to active and refresh token
	acc.Status = "active"
	acc.UsageCount = 0
	if err := refreshAccountToken(acc); err != nil {
		writeAPI(w, http.StatusInternalServerError, apiResponse{Error: "reset failed: " + err.Error()})
		return
	}

	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "Account reset"})
}

// Global proxy config (mutable via API)
var (
	proxyConfig   = defaultProxyConfig()
	proxyConfigMu sync.Mutex
)

type proxyConfigData struct {
	Strategy string            `json:"strategy"`
	Headers  map[string]string `json:"headers"`
}

func defaultProxyConfig() *proxyConfigData {
	return &proxyConfigData{
		Strategy: "round_robin",
		Headers: map[string]string{
			"HTTP-Referer":       "https://cline.bot",
			"X-Title":            "Cline",
			"User-Agent":         "Cline/4.1.16",
			"X-Core-Version":     "4.1.16",
			"X-Platform-Version": "1.106.0",
			"X-Client-Version":   "4.1.16",
			"X-Platform":         "vscode",
			"X-Client-Type":      "cline-vscode",
		},
	}
}

func getProxyConfig() *proxyConfigData {
	proxyConfigMu.Lock()
	defer proxyConfigMu.Unlock()
	return proxyConfig
}

func setProxyConfig(c *proxyConfigData) {
	proxyConfigMu.Lock()
	defer proxyConfigMu.Unlock()
	proxyConfig = c
}

// GET /admin/api/keys
func handleAdminGetKeys(w http.ResponseWriter, r *http.Request) {
	p := loadPool()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"keys": p.Keys}})
}

// POST /admin/api/keys/generate
func handleAdminGenerateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	key := fmt.Sprintf("cline_%x_%x", time.Now().UnixMilli(), time.Now().UnixNano()%1000000)
	p := loadPool()
	poolMu.Lock()
	p.Keys = append(p.Keys, key)
	poolMu.Unlock()
	savePool()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"key": key}})
}

// POST /admin/api/keys/delete  body: { key }
func handleAdminDeleteKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()
	var req struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}
	p := loadPool()
	poolMu.Lock()
	for i, k := range p.Keys {
		if k == req.Key {
			p.Keys = append(p.Keys[:i], p.Keys[i+1:]...)
			break
		}
	}
	poolMu.Unlock()
	savePool()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "Key deleted"})
}

// GET /admin/api/config
func handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	cfg := getProxyConfig()
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{
		"address":      listenAddr,
		"strategy":     cfg.Strategy,
		"version":      "go-1.1",
		"poolPath":     poolPath,
		"defaultModel": defaultModel,
		"headers":      cfg.Headers,
	}})
}

// POST /admin/api/config  body: { strategy?, headers? }
func handleAdminUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		Strategy string            `json:"strategy"`
		Headers  map[string]string `json:"headers"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	cfg := getProxyConfig()
	changed := false

	if req.Strategy != "" {
		switch req.Strategy {
		case "round_robin", "fill", "random":
			cfg.Strategy = req.Strategy
			changed = true
		default:
			writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid strategy, must be: round_robin, fill, random"})
			return
		}
	}

	if req.Headers != nil {
		for k, v := range req.Headers {
			cfg.Headers[k] = v
		}
		changed = true
	}

	if changed {
		setProxyConfig(cfg)
	}

	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{
		"strategy": cfg.Strategy,
		"headers":  cfg.Headers,
	}})
}

// GET /admin/api/models
func handleAdminModels(w http.ResponseWriter, r *http.Request) {
	models := make([]map[string]any, 0)
	for _, m := range supportedModels() {
		models = append(models, map[string]any{
			"id":       m.ID,
			"name":     m.Name,
			"provider": m.Provider,
			"category": m.Category,
			"cost":     m.Cost,
			"status":   m.Status,
			"ownedBy":  m.OwnedBy,
		})
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Data: map[string]any{"models": models}})
}

// POST /admin/api/models/official
func handleAdminModelsOfficial(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	data, err := fetchOfficialModels()
	if err != nil {
		writeAPI(w, http.StatusBadGateway, apiResponse{Error: err.Error()})
		return
	}
	supported := supportedModels()
	officialGroups := []map[string]any{
		officialModelGroup("recommended", "官方推荐", data.Recommended, supported),
		officialModelGroup("free", "免费", data.Free, supported),
		officialModelGroup("clinePass", "Cline Pass", data.Pass, supported),
		officialModelGroup("clineCloud", "Cline Cloud", data.Cloud, supported),
	}
	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Data: map[string]any{
			"source": officialModelsURL,
			"groups": officialGroups,
		},
	})
}

// POST /admin/api/models/add  body: { id, name?, category?, provider? }
func handleAdminModelsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "model id is required"})
		return
	}

	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = providerFromID(req.ID)
	}
	if provider == "unknown" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "model id must be prefixed with a provider, e.g. z-ai/glm-5.3-flash"})
		return
	}

	entry := ModelEntry{
		ID:       req.ID,
		Name:     req.Name,
		Provider: provider,
		Category: req.Category,
		Cost:     modelCostForCategory(req.Category),
		Status:   "active",
		OwnedBy:  provider,
	}
	if entry.Name == "" {
		entry.Name = req.ID
	}
	if entry.Category == "" {
		entry.Category = "custom"
	}

	if err := addSupportedModel(entry); err != nil {
		writeAPI(w, http.StatusConflict, apiResponse{Error: err.Error()})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "Model added: " + entry.ID})
}

// POST /admin/api/models/delete  body: { id }
func handleAdminModelsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "model id is required"})
		return
	}
	if err := removeSupportedModel(req.ID); err != nil {
		writeAPI(w, http.StatusNotFound, apiResponse{Error: err.Error()})
		return
	}
	writeAPI(w, http.StatusOK, apiResponse{Success: true, Message: "Model removed: " + req.ID})
}

// GET /admin/api/stats
func handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}

	p := loadPool()
	active, cooldown, expired := 0, 0, 0
	for _, a := range p.Accounts {
		switch a.Status {
		case "active":
			active++
		case "cooldown":
			cooldown++
		case "expired":
			expired++
		}
	}

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Data: map[string]any{
			"total":    len(p.Accounts),
			"active":   active,
			"cooldown": cooldown,
			"expired":  expired,
			"strategy": "round_robin",
			"version":  "go-1.1",
		},
	})
}

// Authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := loadConfig()

		// If no password is set, allow access
		if cfg.AdminPassword == "" {
			next(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAPI(w, http.StatusUnauthorized, apiResponse{Error: "unauthorized"})
			return
		}

		// Expect "Bearer <password>"
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == cfg.AdminPassword {
			next(w, r)
			return
		}

		writeAPI(w, http.StatusUnauthorized, apiResponse{Error: "invalid password"})
	}
}

// POST /admin/api/auth/verify  body: { password }
func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}

	cfg := loadConfig()

	// If no password is set, always succeed
	if cfg.AdminPassword == "" {
		writeAPI(w, http.StatusOK, apiResponse{
			Success: true,
			Message: "no password required",
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	if req.Password == cfg.AdminPassword {
		writeAPI(w, http.StatusOK, apiResponse{
			Success: true,
			Message: "authenticated",
		})
		return
	}

	writeAPI(w, http.StatusUnauthorized, apiResponse{
		Success: false,
		Error:   "invalid password",
	})
}

// GET /admin/api/auth/check
func handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	cfg := loadConfig()
	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Data: map[string]any{
			"passwordRequired": cfg.AdminPassword != "",
		},
	})
}

// GET /admin/api/scheduler/config
func handleSchedulerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}

	cfg := loadConfig()
	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Data: map[string]any{
			"enabled": cfg.AutoRefreshEnabled,
			"cron":    cfg.AutoRefreshCron,
			"running": scheduler != nil && scheduler.running,
		},
	})
}

// POST /admin/api/scheduler/update  body: { enabled, cron }
func handleSchedulerUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	var req struct {
		Enabled *bool   `json:"enabled"`
		Cron    string `json:"cron"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPI(w, http.StatusBadRequest, apiResponse{Error: "invalid JSON"})
		return
	}

	cfg := loadConfig()
	changed := false

	if req.Enabled != nil {
		cfg.AutoRefreshEnabled = *req.Enabled
		changed = true
	}

	if req.Cron != "" {
		cfg.AutoRefreshCron = req.Cron
		changed = true
	}

	if changed {
		if err := saveConfig(cfg); err != nil {
			writeAPI(w, http.StatusInternalServerError, apiResponse{Error: "failed to save config: " + err.Error()})
			return
		}

		// Restart scheduler with new config
		stopAutoRefresh()
		if cfg.AutoRefreshEnabled {
			startAutoRefresh()
		}
	}

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: "scheduler config updated",
		Data: map[string]any{
			"enabled": cfg.AutoRefreshEnabled,
			"cron":    cfg.AutoRefreshCron,
		},
	})
}

// POST /admin/api/scheduler/trigger
func handleSchedulerTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeAPI(w, http.StatusMethodNotAllowed, apiResponse{Error: "method not allowed"})
		return
	}

	go refreshExpiredAccounts()

	writeAPI(w, http.StatusOK, apiResponse{
		Success: true,
		Message: "manual refresh triggered",
	})
}
