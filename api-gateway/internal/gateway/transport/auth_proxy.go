package transport

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type AuthProxy struct {
	proxy *httputil.ReverseProxy
}

func NewAuthProxy(authServiceURL string) (*AuthProxy, error) {
	target, err := url.Parse(authServiceURL)
	if err != nil {
		return nil, err
	}
	return &AuthProxy{
		proxy: httputil.NewSingleHostReverseProxy(target),
	}, nil
}

func (p *AuthProxy) Register(c *gin.Context) { p.proxy.ServeHTTP(c.Writer, c.Request)}
func (p *AuthProxy) Login(c *gin.Context) { p.proxy.ServeHTTP(c.Writer, c.Request)}
func (p *AuthProxy) Refresh(c *gin.Context) { p.proxy.ServeHTTP(c.Writer, c.Request)}
func (p *AuthProxy) Logout(c *gin.Context) { p.proxy.ServeHTTP(c.Writer, c.Request)}
func (p *AuthProxy) Search(c *gin.Context) { p.proxy.ServeHTTP(c.Writer, c.Request)}
func (p *AuthProxy) ChangePassword(c *gin.Context) { p.proxy.ServeHTTP(c.Writer, c.Request)}