package middlewares

import (
	"SafeBox/services"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type QuotaMiddleware struct {
	quotaService services.QuotaServiceInterface
	redisClient  *redis.Client
}

func NewQuotaMiddleware(qs services.QuotaServiceInterface, rc *redis.Client) *QuotaMiddleware {
	return &QuotaMiddleware{
		quotaService: qs,
		redisClient:  rc,
	}
}

func (m *QuotaMiddleware) EnforceQuota(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method != http.MethodPost {
			return next(c)
		}

		userID := c.Get("userID").(uint)
		contentLength := c.Request().ContentLength

		// Usar Redis Lock para operações concorrentes
		lockKey := fmt.Sprintf("quota_lock:%d", userID)
		lock := m.redisClient.SetNX(c.Request().Context(), lockKey, "locked", 5*time.Second)
		if !lock.Val() {
			return echo.NewHTTPError(http.StatusTooManyRequests, "Concurrent upload in progress")
		}
		defer m.redisClient.Del(c.Request().Context(), lockKey)

		// Verificar cota usando cache
		err := m.quotaService.CheckAndReserveSpace(c.Request().Context(), userID, contentLength)
		if err != nil {
			return echo.NewHTTPError(http.StatusForbidden, err.Error())
		}

		// Rollback em caso de erro
		defer func() {
			if c.Response().Status >= 400 {
				m.quotaService.RollbackSpaceReservation(c.Request().Context(), userID, contentLength)
			}
		}()

		return next(c)
	}
}
