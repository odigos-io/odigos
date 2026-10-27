package middlewares

import (
	"context"
	"crypto/sha256"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	"golang.org/x/time/rate"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ProxyTokenHeader carries central-proxy's projected ServiceAccount token, which
	// proves that the admin override header was set by central-proxy.
	ProxyTokenHeader = "X-Odigos-Proxy-Token"

	proxyTokenAudience      = "odigos-ui"
	proxyTokenReviewTimeout = 5 * time.Second
	proxyTokenCacheTTL      = time.Minute
	proxyTokenMaxLength     = 4096
)

// Successful reviews keyed by the token's SHA-256, never the raw token.
var (
	proxyTokenCacheMu sync.Mutex
	proxyTokenCache   = map[[sha256.Size]byte]time.Time{}

	// Caps TokenReviews from uncached tokens so a flood of forged tokens can't
	// exhaust the UI's own Kubernetes client rate limit.
	proxyTokenReviewLimiter = rate.NewLimiter(10, 20)
)

// WithAdminOverrideFromRequest marks the request context with the admin override
// when the request carries the override header and a central-proxy token that
// Kubernetes authenticates.
func WithAdminOverrideFromRequest(c *gin.Context) {
	if c.GetHeader(AdminOverrideHeader) != "true" {
		return
	}
	if !isCentralProxyToken(c.Request.Context(), c.GetHeader(ProxyTokenHeader)) {
		return
	}
	c.Request = c.Request.WithContext(WithAdminOverride(c.Request.Context()))
}

func isCentralProxyToken(ctx context.Context, token string) bool {
	// A ServiceAccount token is a JWT: three dot-separated parts.
	if token == "" || len(token) > proxyTokenMaxLength || strings.Count(token, ".") != 2 {
		return false
	}
	key := sha256.Sum256([]byte(token))
	now := time.Now()

	proxyTokenCacheMu.Lock()
	expiry, ok := proxyTokenCache[key]
	proxyTokenCacheMu.Unlock()
	if ok && now.Before(expiry) {
		return true
	}

	if !proxyTokenReviewLimiter.Allow() {
		log.Printf("central-proxy token review skipped: too many reviews")
		return false
	}
	if !reviewCentralProxyToken(ctx, token) {
		return false
	}

	proxyTokenCacheMu.Lock()
	for k, exp := range proxyTokenCache {
		if !now.Before(exp) {
			delete(proxyTokenCache, k)
		}
	}
	proxyTokenCache[key] = now.Add(proxyTokenCacheTTL)
	proxyTokenCacheMu.Unlock()
	return true
}

func reviewCentralProxyToken(ctx context.Context, token string) bool {
	if kube.DefaultClient == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, proxyTokenReviewTimeout)
	defer cancel()

	review, err := kube.DefaultClient.AuthenticationV1().TokenReviews().Create(ctx, &authenticationv1.TokenReview{
		Spec: authenticationv1.TokenReviewSpec{Token: token, Audiences: []string{proxyTokenAudience}},
	}, metav1.CreateOptions{})
	if err != nil {
		log.Printf("central-proxy token review failed: %v", err)
		return false
	}

	status := review.Status
	if !status.Authenticated || status.Error != "" {
		return false
	}
	// An audience-aware authenticator echoes the requested audience; without it the
	// token may have been issued for another audience.
	if !slices.Contains(status.Audiences, proxyTokenAudience) {
		return false
	}
	return status.User.Username == "system:serviceaccount:"+env.GetCurrentNamespace()+":"+k8sconsts.CentralProxyServiceAccountName
}

// ReadonlyGuardMiddleware rejects state-changing REST requests in read-only mode,
// the REST counterpart of the GraphQL mutation check in OperationInterceptor.
func ReadonlyGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		WithAdminOverrideFromRequest(c)
		ctx := c.Request.Context()
		if services.IsReadonlyMode(ctx) && !AdminOverrideFromContext(ctx) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": services.ErrorIsReadonly.Error()})
			return
		}
		c.Next()
	}
}
