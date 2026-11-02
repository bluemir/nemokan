package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/location"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionCookieOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	app.Use(location.Default(), fixURL)
	app.Use(sessions.Sessions("test_session", cookie.NewStore([]byte("secret"))), sessionCookieOptions)
	app.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user", "alice")
		require.NoError(t, session.Save())
	})

	for _, testCase := range []struct {
		name     string
		header   string
		isSecure bool
	}{
		{"http", "", false},
		{"https through a proxy", "https", true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/login", nil)
			if testCase.header != "" {
				request.Header.Set("X-Forwarded-Proto", testCase.header)
			}
			recorder := httptest.NewRecorder()
			app.ServeHTTP(recorder, request)

			cookies := recorder.Result().Cookies()
			require.Len(t, cookies, 1)
			assert.True(t, cookies[0].HttpOnly)
			assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
			assert.Equal(t, "/", cookies[0].Path)
			assert.Equal(t, testCase.isSecure, cookies[0].Secure)
		})
	}
}
