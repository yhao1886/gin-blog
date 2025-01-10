package controllers

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const (
	langCookieName     = "lang"
	cookieMaxAge       = 365 * 24 * 60 * 60
	defaultLang        = "en"
	i18nContextKey     = "i18n"
	translationsFolder = "i18n"
)

var (
	bundle       *i18n.Bundle
	translations map[string]map[string]string
)

func init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	loadTranslations()
}

func loadTranslations() {
	translations = make(map[string]map[string]string)
	files, err := os.ReadDir(translationsFolder)
	if err != nil {
		slog.Error("Failed to read translations directory", "error", err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			langCode := file.Name()[:len(file.Name())-5]
			filePath := filepath.Join(translationsFolder, file.Name())
			
			if _, err := bundle.LoadMessageFile(filePath); err != nil {
				slog.Error("Failed to load translation file", "file", file.Name(), "error", err)
				continue
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				slog.Error("Failed to read translation file", "file", file.Name(), "error", err)
				continue
			}

			var langMap map[string]interface{}
			if err := json.Unmarshal(content, &langMap); err != nil {
				slog.Error("Failed to parse translation file", "file", file.Name(), "error", err)
				continue
			}
			translations[langCode] = flattenMap(langMap)
		}
	}
}

func flattenMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	flattenMapRecursive(m, "", result)
	return result
}

func flattenMapRecursive(m map[string]interface{}, prefix string, result map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			flattenMapRecursive(val, key, result)
		case string:
			result[key] = val
		}
	}
}

func GetCurrentLanguage(c *gin.Context) string {
	lang, exists := c.Get(i18nContextKey)
	if !exists {
		return defaultLang
	}
	return lang.(string)
}

func GetTranslation(c *gin.Context, key string) string {
	lang := GetCurrentLanguage(c)
	if trans, ok := translations[lang]; ok {
		if val, exists := trans[key]; exists {
			return val
		}
	}
	
	// Fallback to default language
	if trans, ok := translations[defaultLang]; ok {
		if val, exists := trans[key]; exists {
			return val
		}
	}
	
	return key
}

func SetLanguage(c *gin.Context) {
	lang := c.Query("lang")
	if lang == "" {
		lang = defaultLang
	}

	// Validate language is supported
	if _, exists := translations[lang]; !exists {
		c.JSON(400, gin.H{"error": "Unsupported language"})
		return
	}

	// Set language cookie
	c.SetCookie(langCookieName, lang, cookieMaxAge, "/", "", false, true)
	
	// Update context
	c.Set(i18nContextKey, lang)

	// Redirect back to previous page
	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	
	c.JSON(200, gin.H{
		"status": "success",
		"lang": lang,
		"redirect": referer,
	})
}

func TranslateFunc(c *gin.Context) func(key string) string {
	return func(key string) string {
		return GetTranslation(c, key)
	}
}

// Add to template functions map
func RegisterI18nTemplateFunc(c *gin.Context, h gin.H) {
	h["t"] = TranslateFunc(c)
	h["currentLang"] = GetCurrentLanguage(c)
}