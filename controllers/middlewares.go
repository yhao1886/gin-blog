package controllers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/denisbakhtin/ginblog/config"
	"github.com/denisbakhtin/ginblog/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

//ContextData stores in gin context the common data, such as user info...
func ContextData() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		if uID := session.Get(userIDKey); uID != nil {
			user := models.User{}
			models.GetDB().First(&user, uID)
			if user.ID != 0 {
				c.Set("User", &user)
			}
		}

		if config.GetConfig().SignupEnabled {
			c.Set("SignupEnabled", true)
		}
		c.Next()
	}
}

//AuthRequired grants access to authenticated users, requires SharedData middleware
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, _ := c.Get("User"); user != nil {
			c.Next()
		} else {
			c.Redirect(http.StatusFound, fmt.Sprintf("/signin?return=%s", url.QueryEscape(c.Request.RequestURI)))
			c.Abort()
		}
	}
}

// I18nMiddleware handles language selection and i18n context setup
func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get language from cookie
		lang, err := c.Cookie(langCookieName)
		if err != nil || lang == "" {
			// If no cookie found, use default language from config
			lang = config.GetConfig().DefaultLanguage
			
			// Set cookie with default language
			c.SetCookie(langCookieName, lang, cookieMaxAge, "/", "", false, true)
		}

		// Validate language is supported
		supported := false
		for _, supportedLang := range config.GetConfig().SupportedLanguages {
			if lang == supportedLang {
				supported = true
				break
			}
		}
		if !supported {
			lang = config.GetConfig().DefaultLanguage
		}

		// Store language in context
		c.Set(i18nContextKey, lang)

		// Create localizer for the selected language
		localizer := i18n.NewLocalizer(bundle, lang)
		c.Set("localizer", localizer)

		// Add translation helper to the context
		c.Set("T", TranslateFunc(c))
		c.Next()
	}
}