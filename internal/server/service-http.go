package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gin-contrib/location"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/bluemir/nemokan/internal/buildinfo"
	"github.com/bluemir/nemokan/internal/server/graceful"
	"github.com/bluemir/nemokan/internal/server/injector"
	"github.com/bluemir/nemokan/internal/server/middleware/cache"
	"github.com/bluemir/nemokan/internal/server/middleware/errs"
	"github.com/bluemir/nemokan/internal/server/middleware/prom"
)

func (server *Server) RunServiceHTTPServer(ctx context.Context, bind string, tlsConf *tls.Config) func() error {
	return func() error {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			gin.SetMode(gin.DebugMode)
		} else {
			gin.SetMode(gin.ReleaseMode)
		}

		// starting http server
		app := gin.New()

		// add template
		if html, err := NewRenderer(); err != nil {
			return err
		} else {
			app.SetHTMLTemplate(html)
		}

		// setup Logger
		writer := logrus.
			WithFields(logrus.Fields{}).
			WriterLevel(logrus.InfoLevel)
		defer writer.Close()
		app.Use(gin.LoggerWithWriter(writer))

		// error handler
		app.Use(errs.Middleware)
		app.Use(gin.Recovery())

		// The session cookie options need the scheme, which location reads.
		app.Use(location.Default(), fixURL)

		// sessions
		store := cookie.NewStore([]byte(buildinfo.Signature()))
		app.Use(sessions.Sessions("nemokan_session", store), sessionCookieOptions)
		app.Use(cache.CacheBusting)

		app.Use(injector.Inject(server.backends, server.mail))

		// prometheus for monitoring
		app.Use(prom.Metrics())

		// handle routes
		server.routes(app, app.NoRoute)

		return graceful.Run(ctx, &http.Server{
			Addr:              bind,
			Handler:           app,
			ReadHeaderTimeout: 1 * time.Minute,
			WriteTimeout:      3 * time.Minute,
			IdleTimeout:       3 * time.Minute,

			TLSConfig: tlsConf,
		})
	}
}
