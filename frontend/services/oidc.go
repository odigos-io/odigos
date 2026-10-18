package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

func EnsureOidcSecret(ctx context.Context, secretValue string) error {
	odigosNs := env.GetCurrentNamespace()

	// get existing secret, do not throw on not found
	secret, err := kube.DefaultClient.CoreV1().Secrets(odigosNs).Get(ctx, consts.OidcSecretName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if secret == nil || apierrors.IsNotFound(err) {
		// if the secret doesn't exist, create it
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: consts.OidcSecretName},
			Data:       map[string][]byte{consts.OidcClientSecretProperty: []byte(secretValue)},
		}
		_, err = kube.DefaultClient.CoreV1().Secrets(odigosNs).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		// if the secret exists, update it
		secret.Data[consts.OidcClientSecretProperty] = []byte(secretValue)
		_, err = kube.DefaultClient.CoreV1().Secrets(odigosNs).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func GetOidcSecret(ctx context.Context) (string, error) {
	odigosNs := env.GetCurrentNamespace()
	secret, err := kube.DefaultClient.CoreV1().Secrets(odigosNs).Get(ctx, consts.OidcSecretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(secret.Data[consts.OidcClientSecretProperty]), nil
}

// oidcSettingsUnavailable marks a failure to *read* the OIDC settings from the
// cluster, as opposed to settings that were read and are incomplete. Every request
// that reaches the OIDC middleware reads the effective-config ConfigMap and the
// client secret, so these reads fail whenever the Kubernetes API is briefly
// unavailable. Incomplete settings stay a hard failure.
type oidcSettingsUnavailable struct{ err error }

func (e oidcSettingsUnavailable) Error() string { return e.err.Error() }
func (e oidcSettingsUnavailable) Unwrap() error { return e.err }

// oidcSettings is the OIDC configuration as last read from the cluster, including
// the "no oidc block, authentication is off" outcome.
type oidcSettings struct {
	uiRemoteUrl  string
	tenantUrl    string
	clientId     string
	clientSecret string
	configured   bool
}

var (
	lastOidcSettingsMu sync.RWMutex
	lastOidcSettings   *oidcSettings
)

// resolveOidcSettings reads the OIDC settings, falling back to the settings last
// read successfully when the cluster cannot be reached. Authentication still fails
// closed until the settings have been resolved at least once, so a UI that has
// never read them does not serve requests anonymously.
func resolveOidcSettings(ctx context.Context) (*oidcSettings, error) {
	uiRemoteUrl, tenantUrl, clientId, clientSecret, configured, err := getOidcValuesFromConfig(ctx)
	if err == nil {
		settings := &oidcSettings{
			uiRemoteUrl:  uiRemoteUrl,
			tenantUrl:    tenantUrl,
			clientId:     clientId,
			clientSecret: clientSecret,
			configured:   configured,
		}
		lastOidcSettingsMu.Lock()
		lastOidcSettings = settings
		lastOidcSettingsMu.Unlock()
		return settings, nil
	}

	var unavailable oidcSettingsUnavailable
	if !errors.As(err, &unavailable) {
		return nil, err
	}

	lastOidcSettingsMu.RLock()
	settings := lastOidcSettings
	lastOidcSettingsMu.RUnlock()
	if settings == nil {
		return nil, err
	}

	log.Printf("OIDC: reusing the settings last read from the cluster, reading them failed: %v", err)
	return settings, nil
}

// gets the OIDC configuration values from the odigos-configuration ConfigMap.
// this runs on every request that reaches the OIDC middleware, so failures are
// returned to the caller and must never terminate the process.
func getOidcValuesFromConfig(ctx context.Context) (string, string, string, string, bool, error) {
	if kube.DefaultClient == nil {
		return "", "", "", "", false, oidcSettingsUnavailable{fmt.Errorf("Kubernetes client is not ready")}
	}
	var odigosConfiguration common.OdigosConfiguration
	odigosns := env.GetCurrentNamespace()

	// Allow effective-config reconciliation to finish, bounded by request cancellation.
	effectiveConfigBackoff := wait.Backoff{
		Duration: 2 * time.Second,
		Factor:   2,
		Cap:      16 * time.Second,
		Steps:    6,
	}
	var configMap *corev1.ConfigMap
	err := wait.ExponentialBackoffWithContext(ctx, effectiveConfigBackoff, func(ctx context.Context) (bool, error) {
		var e error
		configMap, e = kube.DefaultClient.CoreV1().ConfigMaps(odigosns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
		if apierrors.IsNotFound(e) {
			return false, nil
		}
		return e == nil, e
	})
	if err != nil {
		return "", "", "", "", false, oidcSettingsUnavailable{fmt.Errorf("error getting effective config CM: %w", err)}
	}
	configYAML, ok := configMap.Data[consts.OdigosConfigurationFileName]
	if !ok || strings.TrimSpace(configYAML) == "" {
		return "", "", "", "", false, fmt.Errorf("effective config is missing %s", consts.OdigosConfigurationFileName)
	}
	err = yaml.Unmarshal([]byte(configYAML), &odigosConfiguration)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("error parsing effective config YAML: %w", err)
	}
	if odigosConfiguration.Oidc == nil {
		return "", "", "", "", false, nil
	}

	if strings.TrimSpace(odigosConfiguration.Oidc.TenantUrl) == "" || strings.TrimSpace(odigosConfiguration.Oidc.ClientId) == "" {
		return "", "", "", "", false, fmt.Errorf("OIDC configuration requires tenantUrl and clientId")
	}

	// UI values
	uiRemoteUrl := odigosConfiguration.UiRemoteUrl
	if uiRemoteUrl == "" {
		uiRemoteUrl = "http://localhost:3000"
	}
	if !strings.HasSuffix(uiRemoteUrl, "/auth/oidc-callback") {
		uiRemoteUrl = fmt.Sprintf("%s/auth/oidc-callback", uiRemoteUrl)
	}

	// An OIDC block enables authentication; missing credentials must not disable it.
	oidcClientSecret, err := GetOidcSecret(ctx)
	if err != nil {
		secretErr := fmt.Errorf("error getting OIDC client secret: %w", err)
		// An absent secret means OIDC is configured but its credentials are not,
		// which must keep failing closed rather than reusing earlier settings.
		if apierrors.IsNotFound(err) {
			return "", "", "", "", false, secretErr
		}
		return "", "", "", "", false, oidcSettingsUnavailable{secretErr}
	}
	if strings.TrimSpace(oidcClientSecret) == "" {
		return "", "", "", "", false, fmt.Errorf("OIDC client secret is empty")
	}
	oidcClientId := odigosConfiguration.Oidc.ClientId
	oidcTenantUrl := odigosConfiguration.Oidc.TenantUrl
	if !strings.HasPrefix(oidcTenantUrl, "https://") {
		oidcTenantUrl = fmt.Sprintf("https://%s", oidcTenantUrl)
	}

	return uiRemoteUrl, oidcTenantUrl, oidcClientId, oidcClientSecret, true, nil
}

func getOidcProvider(ctx context.Context) (string, string, string, string, *oidc.Provider, error) {
	settings, err := resolveOidcSettings(ctx)
	if err != nil {
		return "", "", "", "", nil, err
	}
	if !settings.configured {
		return "", "", "", "", nil, nil
	}

	// Create a Provider through discovery
	oidcProvider, err := oidc.NewProvider(ctx, settings.tenantUrl)
	if err != nil {
		return "", "", "", "", nil, fmt.Errorf("error initializing OIDC provider: %w", err)
	}

	return settings.uiRemoteUrl, settings.tenantUrl, settings.clientId, settings.clientSecret, oidcProvider, nil
}

// GetOidcAuthConfig reads one configuration snapshot for the verifier and OAuth2 flow.
func GetOidcAuthConfig(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	uiRemoteUrl, _, clientID, clientSecret, provider, err := getOidcProvider(ctx)
	if err != nil {
		return nil, nil, err
	}
	if provider == nil {
		return nil, nil, nil
	}
	cfg := &oauth2.Config{
		Endpoint:     provider.Endpoint(),
		RedirectURL:  uiRemoteUrl,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return cfg, provider.Verifier(&oidc.Config{ClientID: clientID}), nil
}

func GetOidcTokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	_, verifier, err := GetOidcAuthConfig(ctx)
	return verifier, err
}

func GetOidcOauthConfig(ctx context.Context) (*oauth2.Config, error) {
	cfg, _, err := GetOidcAuthConfig(ctx)
	return cfg, err
}

func RedirectToOidcAuth(c *gin.Context, oauth2Config *oauth2.Config) {
	state, err := randString(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	nonce, err := randString(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	setCallbackCookie(c.Writer, c.Request, "state", state, "", 0)
	setCallbackCookie(c.Writer, c.Request, "nonce", nonce, "", 0)

	c.Redirect(http.StatusFound, oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)))
}

func OidcAuthCallback(ctx context.Context, c *gin.Context) {
	requestCtx, cancel := OidcRequestContext(ctx, c.Request.Context())
	defer cancel()
	ctx = requestCtx
	oauth2Config, oidcTokenVerifier, err := GetOidcAuthConfig(ctx)
	if err != nil {
		log.Printf("OIDC callback configuration unavailable: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "OIDC configuration is unavailable"})
		return
	}
	// We're in a callback (after being redirected from auth),
	// so we should always have OIDC & OAuth2 configured here.
	if oidcTokenVerifier == nil || oauth2Config == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "OIDC or OAuth2 is not configured"})
		return
	}

	urlQuery := c.Request.URL.Query()

	// Verify the 'state' from the query parameters against the cookies
	state, err := c.Cookie("state")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"state": err.Error()})
		return
	}
	if urlQuery.Get("state") != state {
		c.JSON(http.StatusInternalServerError, gin.H{"state": "mismatch"})
		return
	}

	// Exchange the authorization code for an OAuth2 token
	code := urlQuery.Get("code")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"token exchange": err.Error()})
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"id_token": "missing in oauth2 token"})
		return
	}
	idToken, err := oidcTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"token verify": err.Error()})
		return
	}

	// Verify the 'nonce' from the cookies against the token payload
	nonce, err := c.Cookie("nonce")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"nonce": err.Error()})
		return
	}
	if idToken.Nonce != nonce {
		c.JSON(http.StatusInternalServerError, gin.H{"nonce": "mismatch"})
		return
	}

	setCallbackCookie(c.Writer, c.Request, "id_token", rawIDToken, "/", int(idToken.Expiry.Unix()-idToken.IssuedAt.Unix()))

	c.Redirect(http.StatusFound, "/")
}

// OidcRequestContext bounds authentication work and cancels it when the client disconnects.
func OidcRequestContext(serverCtx, requestCtx context.Context) (context.Context, context.CancelFunc) {
	client := *http.DefaultClient
	if configured, ok := serverCtx.Value(oauth2.HTTPClient).(*http.Client); ok && configured != nil {
		client = *configured
	}
	timeout := 10 * time.Second
	if client.Timeout > 0 && client.Timeout < timeout {
		timeout = client.Timeout
	}
	ctx, cancel := context.WithTimeout(serverCtx, timeout)
	stop := context.AfterFunc(requestCtx, cancel)
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	// go-oidc detaches JWKS fetches from verifier cancellation. Bind the HTTP
	// transport as well so disconnects stop the underlying network request.
	client.Transport = oidcRequestTransport{ctx: ctx, base: transport}
	return oidc.ClientContext(ctx, &client), func() { stop(); cancel() }
}

type oidcRequestTransport struct {
	ctx  context.Context
	base http.RoundTripper
}

func (t oidcRequestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.base.RoundTrip(req.WithContext(t.ctx))
}
