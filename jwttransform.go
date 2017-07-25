package jwttransform

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

// JWTTransform ...
type JWTTransform struct {
	Next     httpserver.Handler
	Upstream string
}

type Config struct {
	UpstreamURL string
}

type Target struct {
	Token     string            `json:"token"`
	TokenType string            `json:"token_type"`
	TokenExp  int               `json:"token_exp"`
	Headers   map[string]string `json:"headers"`
}

func parseConfig(c *caddy.Controller) Config {

	// var upstream string

	conf := Config{}

	for c.Next() {
		token := c.Val()
		args := c.RemainingArgs()

		switch token {
		case "upstream":
			conf.UpstreamURL = args[0]
		}

	}

	return conf
}

func (t JWTTransform) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {

	resp, err := http.Get(t.Upstream)
	if err != nil {
		panic(err)
	}

	var target Target

	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&target)
	auth := r.Header.Get("Authorization")

	if auth != "" {
		r.Header.Set("Authorization", "FOOBAR BARFOO")
	}

	return t.Next.ServeHTTP(w, r)
}

func setup(c *caddy.Controller) error {

	cfg := httpserver.GetConfig(c)

	config := parseConfig(c)

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return JWTTransform{Next: next, Upstream: config.UpstreamURL}
	})
	return nil
}

func init() {
	fmt.Println("Hello Init JWTTransform")
	caddy.RegisterPlugin("jwttransform", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
	httpserver.RegisterDevDirective("jwttransform", "jwt")
}
