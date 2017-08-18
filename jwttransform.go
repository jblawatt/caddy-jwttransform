package jwttransform

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

// JWTTransform ...
type JWTTransform struct {
	Next       httpserver.Handler
	Config     Config
	TokenCache *cache.Cache
}

// https://www.oauth.com/oauth2-servers/authorization/the-authorization-request/
// https://aaronparecki.com/oauth-2-simplified/

type Config struct {
	LoginPath           string
	AuthURL             string
	Scope               string
	State               string
	Headers             map[string]string
	ClientID            string
	ClientSecretHeader  string
	AuthorizationHeader string
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	State        string `json:"state"`
}

// TODO: Unsuccessful response

func parseConfig(c *caddy.Controller) Config {
	conf := Config{
		ClientSecretHeader:  "X-Client-Secret",
		AuthorizationHeader: "Authorization",
		LoginPath:           "/",
	}

	for c.Next() {

		token := c.Val()
		args := c.RemainingArgs()

		switch token {
		case "login_path":
			conf.LoginPath = args[0]
		case "auth_url":
			conf.AuthURL = args[0]
		// case "scope":
		// 	conf.Scope = args[0]
		// case "state":
		// 	conf.State = args[0]
		case "client_id":
			conf.ClientID = args[0]
		case "client_secret_header":
			conf.ClientSecretHeader = args[0]
		case "auth_header":
			conf.AuthorizationHeader = args[0]
		}

	}

	return conf
}

func (t JWTTransform) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	if !(strings.HasPrefix(r.URL.Path, t.Config.LoginPath)) {
		return t.Next.ServeHTTP(w, r)
	}

	clientSecret := r.Header.Get(t.Config.ClientSecretHeader)
	if clientSecret == "" {
		return http.StatusBadRequest, errors.New("Missing Authorization")
	}

	var tokenResp OAuthTokenResponse

	cacheKey := t.Config.ClientID + "~" + clientSecret
	// hasCacheHit
	cacheHit, _ := t.TokenCache.Get(cacheKey)
	if false {
		tokenResp = cacheHit.(OAuthTokenResponse)
	} else {
		req, _ := http.NewRequest("POST", t.Config.AuthURL, nil)
		query := req.URL.Query()
		query.Add("grant_type", "client_credentials")
		query.Add("client_id", t.Config.ClientID)
		query.Add("client_secret", clientSecret)
		req.URL.RawQuery = query.Encode()

		for key, value := range t.Config.Headers {
			req.Header.Add(key, value)
		}
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&tokenResp)
		t.TokenCache.Set(
			cacheKey,
			tokenResp,
			time.Duration(tokenResp.ExpiresIn-200)*time.Second,
		)
	}
	fmt.Println(tokenResp)
	r.Header.Set(t.Config.AuthorizationHeader, tokenResp.TokenType+" "+tokenResp.AccessToken)

	return t.Next.ServeHTTP(w, r)
}

func setup(c *caddy.Controller) error {

	tokenCache := cache.New(3*time.Minute, 10*time.Minute)

	cfg := httpserver.GetConfig(c)

	config := parseConfig(c)

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return JWTTransform{Next: next, Config: config, TokenCache: tokenCache}
	})
	return nil
}

func init() {
	caddy.RegisterPlugin("jwttransform", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
	httpserver.RegisterDevDirective("jwttransform", "jwt")
	log.Println("[DEBUG] JWT Transform Middleware initialized.")
}
