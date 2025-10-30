package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"lab1/internal/app/ds"
	"log"
	"net/http"
	"strings"
)

const jwtPrefix = "Bearer "

func (h *Handler) WithAuthCheck(role bool) func(ctx *gin.Context) {
	return func(gCtx *gin.Context) {
		jwtStr := gCtx.GetHeader("Authorization")
		if !strings.HasPrefix(jwtStr, jwtPrefix) { // если нет префикса то нас дурят!
			gCtx.AbortWithStatus(http.StatusForbidden) // отдаем что нет доступа

			return // завершаем обработку
		}

		// отрезаем префикс
		jwtStr = jwtStr[len(jwtPrefix):]

		if h.Repository.RedisClient != nil {
			isBlacklisted, err := h.Repository.RedisClient.CheckJWTInBlacklist(gCtx.Request.Context(), jwtStr)
			if err != nil {
				gCtx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Ошибка проверки токена",
				})
				return
			}
			if isBlacklisted {
				gCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Токен заблокирован",
				})
				return
			}
		}

		token, err := jwt.ParseWithClaims(jwtStr, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Config.JWT.Token), nil
		})
		if err != nil {
			gCtx.AbortWithStatus(http.StatusForbidden)
			log.Println(err)

			return
		}

		myClaims := token.Claims.(*ds.JWTClaims)

		var roleUser int
		var roles int

		if role {
			roles = 1 //moderator
		} else {
			roles = 2 //ne moderator a user
		}

		if myClaims.IsAdmin {
			roleUser = 1
		} else {
			roleUser = 2
		}

		if roleUser != roles && roleUser == 2 {
			gCtx.AbortWithStatus(http.StatusForbidden)
			log.Printf("role is not assigned")
			return
		}
		gCtx.Set("userID", myClaims.UserID)
		gCtx.Set("isAdmin", myClaims.IsAdmin)

		gCtx.Next()
	}
}

func (h *Handler) WithAuthCheckLab5(role bool) func(ctx *gin.Context) {
	return func(gCtx *gin.Context) {
		jwtStr := gCtx.GetHeader("Authorization")
		if !strings.HasPrefix(jwtStr, jwtPrefix) { // если нет префикса то нас дурят!
			gCtx.Set("userID", uint(0))
			gCtx.Next()
			return
		}

		// отрезаем префикс
		jwtStr = jwtStr[len(jwtPrefix):]

		if h.Repository.RedisClient != nil {
			isBlacklisted, err := h.Repository.RedisClient.CheckJWTInBlacklist(gCtx.Request.Context(), jwtStr)
			if err != nil {
				gCtx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Ошибка проверки токена",
				})
				return
			}
			if isBlacklisted {
				gCtx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Токен заблокирован",
				})
				return
			}
		}

		token, err := jwt.ParseWithClaims(jwtStr, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Config.JWT.Token), nil
		})
		if err != nil {
			gCtx.AbortWithStatus(http.StatusForbidden)
			log.Println(err)

			return
		}

		myClaims := token.Claims.(*ds.JWTClaims)

		var roleUser int
		var roles int

		if role {
			roles = 1 //moderator
		} else {
			roles = 2 //ne moderator a user
		}

		if myClaims.IsAdmin {
			roleUser = 1
		} else {
			roleUser = 2
		}

		if roleUser != roles && roleUser == 2 {
			gCtx.AbortWithStatus(http.StatusForbidden)
			log.Printf("role is not assigned")
			return
		}
		gCtx.Set("userID", myClaims.UserID)
		gCtx.Set("isAdmin", myClaims.IsAdmin)

		gCtx.Next()
	}
}
