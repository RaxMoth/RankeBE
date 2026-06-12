package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"ranke-be/internal/apple"
	db "ranke-be/internal/db/sqlc"
	"ranke-be/internal/middleware"
)

// AppleNotificationHandler receives server-to-server notifications from
// Apple's identity service (consent revoked, account deleted, email
// changed, ...).
//
// This endpoint is intentionally unauthenticated — Apple signs the
// payload as a JWT, and the verifier validates issuer/audience/signature
// against Apple's JWKS. Anyone can POST garbage; only payloads that
// verify against Apple's current keys are acted on.
//
// Register the URL ("Server-to-Server Notification Endpoint") in the
// Apple Developer portal for your Services ID — Apple will not call this
// otherwise.
type AppleNotificationHandler struct {
	verifier *apple.Verifier
	queries  *db.Queries
	logger   *slog.Logger
}

func NewAppleNotificationHandler(verifier *apple.Verifier, queries *db.Queries, logger *slog.Logger) *AppleNotificationHandler {
	return &AppleNotificationHandler{verifier: verifier, queries: queries, logger: logger}
}

// notificationRequest mirrors Apple's wire shape — a single JSON object
// with one field, `payload`, whose value is the signed JWT.
type notificationRequest struct {
	Payload string `json:"payload" binding:"required"`
}

func (h *AppleNotificationHandler) Handle(c *gin.Context) {
	if h.verifier == nil {
		// Apple Sign-In not configured for this deployment. We accept the
		// POST quietly with 200 so Apple doesn't keep retrying — there is
		// no way for them to know we're not configured.
		c.Status(http.StatusOK)
		return
	}

	var req notificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Don't 4xx — Apple's retry logic treats non-2xx as failure and
		// will keep retrying for ~24h. We can't distinguish a malformed
		// payload from a benign one without verifying the JWT first.
		h.logger.Warn("apple notification: malformed body",
			slog.String("request_id", middleware.GetRequestID(c)),
			slog.String("error", err.Error()))
		c.Status(http.StatusOK)
		return
	}

	event, err := h.verifier.VerifyNotification(c.Request.Context(), req.Payload)
	if err != nil {
		h.logger.Warn("apple notification: invalid jwt",
			slog.String("request_id", middleware.GetRequestID(c)),
			slog.String("error", err.Error()))
		// Same reasoning as above — return 200 so Apple stops retrying a
		// payload we can never accept.
		c.Status(http.StatusOK)
		return
	}

	h.logger.Info("apple notification received",
		slog.String("request_id", middleware.GetRequestID(c)),
		slog.String("type", event.Type),
		slog.String("sub", event.Sub))

	switch event.Type {
	case "consent-revoked", "account-delete":
		// User has either revoked our app's access to their Apple ID, or
		// deleted the Apple ID outright. Either way they no longer want
		// us to have their data — delete the local row, cascade does the
		// rest. App Store guideline 5.1.1(v) again.
		user, err := h.queries.GetUserByAppleSub(c.Request.Context(),
			pgtype.Text{String: event.Sub, Valid: true})
		if err != nil {
			// Already deleted, or never signed in via Apple — nothing to do.
			c.Status(http.StatusOK)
			return
		}
		if err := h.queries.DeleteUser(c.Request.Context(), user.ID); err != nil {
			h.logger.Error("apple notification: delete user failed",
				slog.String("request_id", middleware.GetRequestID(c)),
				slog.String("sub", event.Sub),
				slog.String("error", err.Error()))
			// Return 500 so Apple retries — this is a real server failure.
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

	case "email-disabled", "email-enabled":
		// We don't have a separate state for hide-my-email forwarding
		// addresses today; safe to ignore. If/when we add transactional
		// email, route this through to mark the address bouncing.

	default:
		// Unknown event types: log and accept. Apple may add new ones
		// without warning, and 500-ing on them causes retries.
		h.logger.Info("apple notification: unknown event type",
			slog.String("request_id", middleware.GetRequestID(c)),
			slog.String("type", event.Type))
	}

	c.Status(http.StatusOK)
}
