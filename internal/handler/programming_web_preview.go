package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const previewLeaseTTL = 10 * time.Minute

var previewHostPattern = regexp.MustCompile(`^wkpreview-([a-f0-9]{32})\.`)

type previewLease struct {
	UserID    string
	TenantID  uint64
	SessionID string
	SandboxID string
	Port      int
	RelayPort int
	Secret    string
	Host      string
	AppOrigin string
	Expires   time.Time
}
type previewLeaseStore struct {
	redis  *redis.Client
	mu     sync.Mutex
	leases map[string]previewLease
}

func (s *previewLeaseStore) get(ctx context.Context, token string) (previewLease, bool) {
	var lease previewLease
	if s.redis != nil {
		raw, err := s.redis.Get(ctx, "weknora:web-preview:"+token).Bytes()
		if err != nil || json.Unmarshal(raw, &lease) != nil {
			return lease, false
		}
	} else {
		s.mu.Lock()
		lease = s.leases[token]
		s.mu.Unlock()
	}
	return lease, lease.UserID != "" && time.Now().Before(lease.Expires)
}
func (s *previewLeaseStore) put(ctx context.Context, token string, lease previewLease) error {
	lease.Expires = time.Now().Add(previewLeaseTTL)
	if s.redis != nil {
		raw, _ := json.Marshal(lease)
		return s.redis.Set(ctx, "weknora:web-preview:"+token, raw, previewLeaseTTL).Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, item := range s.leases {
		if time.Now().After(item.Expires) {
			delete(s.leases, key)
		}
	}
	s.leases[token] = lease
	return nil
}

// Install before application CORS/auth/routes: a preview origin must NEVER
// fall through to the authenticated application, including on an invalid lease.
func (h *ProgrammingHandler) WebPreviewMiddleware(client *redis.Client) gin.HandlerFunc {
	h.previewLeases = &previewLeaseStore{redis: client, leases: make(map[string]previewLease)}
	return func(c *gin.Context) {
		hostname := strings.ToLower(c.Request.Host)
		if host, _, err := net.SplitHostPort(hostname); err == nil {
			hostname = host
		}
		if !strings.HasPrefix(hostname, "wkpreview-") {
			c.Next()
			return
		}
		c.Abort()
		match := previewHostPattern.FindStringSubmatch(hostname)
		if len(match) != 2 {
			c.Status(http.StatusNotFound)
			return
		}
		lease, ok := h.previewLeases.get(c.Request.Context(), match[1])
		if !ok || !strings.EqualFold(lease.Host, c.Request.Host) {
			c.Status(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(c.Request.Context(), types.UserIDContextKey, lease.UserID)
		ctx = context.WithValue(ctx, types.TenantIDContextKey, lease.TenantID)
		if _, err := h.sessions.GetOwnedSession(ctx, lease.SessionID); err != nil {
			c.Status(http.StatusForbidden)
			return
		}
		proxy := &httputil.ReverseProxy{
			Rewrite: func(p *httputil.ProxyRequest) {
				p.Out.URL.Scheme = "http"
				p.Out.URL.Host = "localhost"
				p.Out.Host = "localhost"
				for _, name := range []string{"Authorization", "X-API-Key", "X-Tenant-ID", "X-Embed-Session", "X-External-User-Token", sandbox.InboundTokenHeader} {
					p.Out.Header.Del(name)
				}
				p.Out.Header.Set("X-WeKnora-Preview-Relay", lease.Secret)
				// Project websocket Origin checks should see the project listener.
				if p.Out.Header.Get("Origin") != "" {
					p.Out.Header.Set("Origin", "http://localhost:"+strconv.Itoa(lease.Port))
				}
			},
			Transport: previewRoundTripper(func(req *http.Request) (*http.Response, error) {
				return h.workspace.RoundTripWebPreview(ctx, lease.SessionID, lease.SandboxID, lease.RelayPort, req)
			}),
			ModifyResponse: func(response *http.Response) error {
				response.Header.Del(sandbox.InboundTokenHeader)
				response.Header.Del("X-WeKnora-Preview-Relay")
				response.Header.Del("X-Frame-Options")
				response.Header.Set("Content-Security-Policy", "frame-ancestors "+lease.AppOrigin+"; sandbox allow-scripts allow-same-origin allow-forms allow-downloads")
				response.Header.Set("Referrer-Policy", "no-referrer")
				response.Header.Set("Cache-Control", "no-store")
				response.Header.Del("Access-Control-Allow-Origin")
				response.Header.Del("Access-Control-Allow-Credentials")
				// Project cookies are restricted to this project's unique origin.
				cookies := response.Cookies()
				response.Header.Del("Set-Cookie")
				for _, cookie := range cookies {
					cookie.Domain = ""
					response.Header.Add("Set-Cookie", cookie.String())
				}
				if location, err := url.Parse(response.Header.Get("Location")); err == nil && location.Host != "" && (location.Hostname() == "localhost" || location.Hostname() == "127.0.0.1") && location.Port() == strconv.Itoa(lease.Port) {
					location.Scheme = ""
					location.Host = ""
					response.Header.Set("Location", location.String())
				}
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
				http.Error(w, "Project preview is unavailable. Reopen Web preview to reconnect.", http.StatusBadGateway)
			},
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<20)
		proxy.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
	}
}

type previewRoundTripper func(*http.Request) (*http.Response, error)

func (f previewRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Public base routes *.base to this application's HTTP listener. Development
// uses *.localhost on the API listener, requiring no DNS or desktop sandbox.
func previewPublicBase(request *http.Request) (*url.URL, error) {
	raw := strings.TrimSpace(os.Getenv("WEKNORA_PREVIEW_BASE_URL"))
	if raw != "" {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, errors.New("invalid preview base")
		}
		u.Path = ""
		return u, nil
	}
	origin, err := url.Parse(request.Header.Get("Origin"))
	if err != nil || origin == nil || (origin.Hostname() != "localhost" && origin.Hostname() != "127.0.0.1" && origin.Hostname() != "::1") || origin.Scheme != "http" {
		return nil, errors.New("preview domain required")
	}
	_, port, err := net.SplitHostPort(request.Host)
	if err != nil {
		port = "80"
	}
	return &url.URL{Scheme: "http", Host: net.JoinHostPort("localhost", port)}, nil
}

func (h *ProgrammingHandler) DiscoverWebPreview(c *gin.Context) {
	if _, ok := h.loadOwnedSession(c); !ok {
		return
	}
	var request struct {
		Start     bool   `json:"start"`
		PreviewID string `json:"preview_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	app, err := url.Parse(c.GetHeader("Origin"))
	if err != nil || app == nil || (app.Scheme != "http" && app.Scheme != "https") || app.Host == "" || app.User != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	base, err := previewPublicBase(c.Request)
	if err != nil {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"status": "setup_required"}})
		return
	}
	ctx := c.Request.Context()
	sessionID := programmingSessionID(c)
	preview, err := h.workspace.DiscoverWebPreview(ctx, sessionID, request.Start)
	if err != nil {
		status := "error"
		if errors.Is(err, sandbox.ErrNoLiveSessionSandbox) {
			status = "not_running"
		}
		if errors.Is(err, sandbox.ErrSandboxPaused) {
			status = "paused"
		}
		if errors.Is(err, sandbox.ErrPreviewUnsupported) {
			status = "unsupported"
		}
		c.JSON(200, gin.H{"success": true, "data": gin.H{"status": status}})
		return
	}
	if preview.Status != "ready" {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"status": preview.Status, "detail": preview.Detail}})
		return
	}
	sandboxID, ok := h.workspace.BoundSandboxID(ctx, sessionID)
	if !ok {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"status": "not_running"}})
		return
	}
	userID, _ := types.UserIDFromContext(ctx)
	tenantID, _ := types.TenantIDFromContext(ctx)
	if userID == "" || tenantID == 0 || h.previewLeases == nil {
		c.Status(http.StatusUnauthorized)
		return
	}
	token := request.PreviewID
	old, found := h.previewLeases.get(ctx, token)
	if !found || old.UserID != userID || old.TenantID != tenantID || old.SessionID != sessionID || old.SandboxID != sandboxID || old.Port != preview.Port || old.Secret != preview.Secret || old.AppOrigin != app.Scheme+"://"+app.Host || !strings.HasSuffix(old.Host, "."+base.Host) {
		var entropy [16]byte
		if _, err := rand.Read(entropy[:]); err != nil {
			c.Status(500)
			return
		}
		token = hex.EncodeToString(entropy[:])
	}
	base.Host = "wkpreview-" + token + "." + base.Host
	lease := previewLease{UserID: userID, TenantID: tenantID, SessionID: sessionID, SandboxID: sandboxID, Port: preview.Port, RelayPort: preview.RelayPort, Secret: preview.Secret, Host: base.Host, AppOrigin: app.Scheme + "://" + app.Host}
	if err := h.previewLeases.put(ctx, token, lease); err != nil {
		c.Status(503)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"status": "ready", "url": base.String() + "/", "preview_id": token, "port": preview.Port}})
}
