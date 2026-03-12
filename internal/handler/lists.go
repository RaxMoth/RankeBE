package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

type ListHandler struct {
	lists   *service.ListService
}

func NewListHandler(lists *service.ListService) *ListHandler {
	return &ListHandler{lists: lists}
}

type createListRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
	ValueType   string  `json:"value_type" binding:"required,oneof=number duration text"`
	RankOrder   string  `json:"rank_order" binding:"required,oneof=asc desc"`
	IsPublic    *bool   `json:"is_public"`
}

func (h *ListHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	var req createListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	desc := pgtype.Text{}
	if req.Description != nil {
		desc = pgtype.Text{String: *req.Description, Valid: true}
	}
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	list, err := h.lists.CreateList(c.Request.Context(), userID, req.Title, desc, req.ValueType, req.RankOrder, isPublic)
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusCreated, list)
}

func (h *ListHandler) GetUserLists(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	rows, err := h.lists.GetUserLists(c.Request.Context(), userID)
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, rows)
}

func (h *ListHandler) GetByID(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	list, err := h.lists.GetListByID(c.Request.Context(), listID)
	if err != nil {
		Fail(c, http.StatusNotFound, CodeListNotFound, "list not found")
		return
	}

	// Access check: public lists visible to any authed user; private only to members
	if !list.IsPublic {
		_, err := h.lists.GetMemberRole(c.Request.Context(), listID, userID)
		if err != nil {
			Forbidden(c)
			return
		}
	}

	entries, err := h.lists.GetRankedEntries(c.Request.Context(), list)
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, gin.H{
		"list":    list,
		"entries": entries,
	})
}

type updateListRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

func (h *ListHandler) Update(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	var req updateListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	desc := pgtype.Text{}
	if req.Description != nil {
		desc = pgtype.Text{String: *req.Description, Valid: true}
	}
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	list, err := h.lists.UpdateList(c.Request.Context(), listID, req.Title, desc, isPublic)
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, list)
}

func (h *ListHandler) Delete(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	if err := h.lists.DeleteList(c.Request.Context(), listID); err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "list deleted"})
}

func (h *ListHandler) JoinPublic(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	if err := h.lists.JoinPublicList(c.Request.Context(), listID, userID); err != nil {
		Fail(c, http.StatusBadRequest, CodeValidationError, err.Error())
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "joined"})
}

func (h *ListHandler) GetInvitePreview(c *gin.Context) {
	tokenStr := c.Param("token")
	token, ok := middleware.ParseUUID(tokenStr)
	if !ok {
		ValidationError(c, "invalid invite token")
		return
	}

	preview, err := h.lists.GetInvitePreview(c.Request.Context(), token)
	if err != nil {
		Fail(c, http.StatusNotFound, CodeListNotFound, "invalid invite link")
		return
	}

	Success(c, http.StatusOK, preview)
}

func (h *ListHandler) JoinByInvite(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	tokenStr := c.Param("token")
	token, ok := middleware.ParseUUID(tokenStr)
	if !ok {
		ValidationError(c, "invalid invite token")
		return
	}

	list, err := h.lists.JoinByInvite(c.Request.Context(), token, userID)
	if err != nil {
		Fail(c, http.StatusBadRequest, CodeValidationError, err.Error())
		return
	}

	Success(c, http.StatusOK, list)
}

func (h *ListHandler) GetMembers(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	members, err := h.lists.GetMembers(c.Request.Context(), listID)
	if err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, members)
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}

func (h *ListHandler) UpdateMemberRole(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	targetUserID, ok := middleware.ParseUUID(c.Param("userId"))
	if !ok {
		ValidationError(c, "invalid user id")
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	member, err := h.lists.UpdateMemberRole(c.Request.Context(), listID, targetUserID, req.Role)
	if err != nil {
		Fail(c, http.StatusBadRequest, CodeValidationError, err.Error())
		return
	}

	Success(c, http.StatusOK, member)
}

func (h *ListHandler) RemoveMember(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	targetUserID, ok := middleware.ParseUUID(c.Param("userId"))
	if !ok {
		ValidationError(c, "invalid user id")
		return
	}

	if err := h.lists.RemoveMember(c.Request.Context(), listID, targetUserID); err != nil {
		Fail(c, http.StatusBadRequest, CodeValidationError, err.Error())
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "member removed"})
}

type rankUpdateItem struct {
	EntryID string `json:"entry_id" binding:"required"`
	Rank    int    `json:"rank" binding:"required"`
}

func (h *ListHandler) BulkUpdateRanks(c *gin.Context) {
	var items []rankUpdateItem
	if err := c.ShouldBindJSON(&items); err != nil {
		ValidationError(c, err.Error())
		return
	}

	updates := make([]service.RankUpdate, 0, len(items))
	for _, item := range items {
		entryID, ok := middleware.ParseUUID(item.EntryID)
		if !ok {
			ValidationError(c, "invalid entry_id: "+item.EntryID)
			return
		}
		updates = append(updates, service.RankUpdate{
			EntryID: entryID,
			Rank:    item.Rank,
		})
	}

	if err := h.lists.BulkUpdateRanks(c.Request.Context(), updates); err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "ranks updated"})
}
