package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	db "ranke-be/internal/db/sqlc"
	"ranke-be/internal/handler/dto"
	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

type UserHandler struct {
	queries *db.Queries
	lists   *service.ListService
}

func NewUserHandler(queries *db.Queries, lists *service.ListService) *UserHandler {
	return &UserHandler{queries: queries, lists: lists}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		NotFound(c, "user not found")
		return
	}

	Success(c, http.StatusOK, dto.MapUser(user, true))
}

type updateMeRequest struct {
	DisplayName string `json:"displayName" binding:"required,min=1"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	user, err := h.queries.UpdateDisplayName(c.Request.Context(), db.UpdateDisplayNameParams{
		ID:          userID,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, dto.MapUser(user, true))
}

// GetUserProfile returns a user's *public* profile — display name, avatar,
// member-since timestamp, and the list of public boards they belong to.
//
// Email is intentionally omitted (privacy). The caller is whoever is authed,
// not the profile subject.
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	userID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid user id")
		return
	}

	profileRow, err := h.queries.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		NotFound(c, "user not found")
		return
	}

	boardRows, err := h.queries.GetPublicBoardsForUser(c.Request.Context(), userID)
	if err != nil {
		InternalError(c)
		return
	}

	boards := make([]dto.ListSummary, 0, len(boardRows))
	for _, r := range boardRows {
		// We could call GetUserCurrentRank for each board to populate
		// `ownRank`, but that's N queries per profile load. Skip it for
		// now — the home feed already has it for the caller's own boards;
		// public profiles render rank-less summaries.
		boards = append(boards, dto.MapPublicBoardsForUserRow(r, nil))
	}

	Success(c, http.StatusOK, dto.MapUserProfile(profileRow, boards))
}
