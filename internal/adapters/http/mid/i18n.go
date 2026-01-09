// Package mid provides HTTP middleware implementations for request processing.
// It includes middleware for internationalization (i18n) that handles locale detection
// from cookies, query parameters, and headers.
package mid

import (
	"net/http"
	"time"

	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/invopop/ctxi18n"
)

const langCookie = "lang"

// I18n is middleware that sets the request context with a locale based on cookies, query parameters, or headers.
func I18n(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var lang string

		if c, err := r.Cookie(langCookie); err == nil {
			lang = c.Value
		}

		if lang == "" {
			lang = r.URL.Query().Get("lang")
		}

		if lang == "" {
			lang = r.Header.Get("Accept-Language")
		}

		ctx, err := ctxi18n.WithLocale(r.Context(), lang)
		if err != nil {
			utils.Logger.Error("failed to set locale", "lang", lang, "err", err)
		}

		if r.URL.Query().Has("lang") {
			http.SetCookie(w, &http.Cookie{
				Name:     langCookie,
				Value:    ctxi18n.Locale(ctx).Code().String(),
				Path:     "/",
				HttpOnly: false,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int((365 * 24 * time.Hour).Seconds()),
			})
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
