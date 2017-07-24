package jwttransform

import (
	"encoding/json"
	"fmt"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"net/http"
)

// JWTTransform ...
type JWTTransform struct {
	Next     httpserver.Handler
	Upstream string
}

type Target struct {
	Token string `json:"token"`
	Type  string `json:"type"`
}

func (t JWTTransform) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	fmt.Println("Hello from Middleware")

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
	fmt.Println("JWTTransform Setup")
	cfg := httpserver.GetConfig(c)

	var upstream string

	for c.Next() {
		for c.NextBlock() {
			upstream = c.Val()
		}
	}

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return JWTTransform{Next: next, Upstream: upstream}
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
