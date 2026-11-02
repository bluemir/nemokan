package server

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/location"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func html(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		markAcceptHTML(c)
		c.HTML(http.StatusOK, path, c)
	}
}
func redirect(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, path)
	}
}

func (server *Server) AbortIfHasPrefix(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, prefix) {
			c.Abort()
		}
	}
}

func fixURL(c *gin.Context) {
	url := location.Get(c)

	// QUESTION is it right?
	c.Request.URL.Scheme = url.Scheme
	c.Request.URL.Host = url.Host
}

// sessionCookieOptions keeps the session cookie from page scripts (HttpOnly) and from requests
// started by other sites (SameSite Lax, so links from other sites still keep the login).
// It is Secure when the request came over HTTPS, directly or through a proxy, so the cookie
// is not sent over plain HTTP after that.
func sessionCookieOptions(c *gin.Context) {
	sessions.Default(c).Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Request.URL.Scheme == "https",
	})
}

func markAcceptJSON(c *gin.Context) {
	c.SetAccepted("application/json")
}
func markAcceptHTML(c *gin.Context) {
	c.SetAccepted("text/html")
}
