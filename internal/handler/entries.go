package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

type EntryHandler struct {
	entries *service.EntryService
	lists   *service.ListService
}

func NewEntryHandler(entries *service.EntryService, lists *service.ListService) *EntryHandler {
	return &EntryHandler{entries: entries, lists: lists}
}

type upsertEntryRequest struct {
	ValueNumber     *float64 `json:"value_number"`
	ValueDurationMs *int64   `json:"value_duration_ms"`
	ValueText       *string  `json:"value_text"`
	Note            *string  `json:"note"`
}

func (h *EntryHandler) UpsertMyEntry(c *gin.Context) {
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

	var req upsertEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	// Get list to know the value_type
	list, err := h.lists.GetListByID(c.Request.Context(), listID)
	if err != nil {
		Fail(c, http.StatusNotFound, CodeListNotFound, "list not found")
		return
	}

	entry, err := h.entries.UpsertEntry(c.Request.Context(), listID, userID, list.ValueType, service.UpsertEntryInput{
		ValueNumber:     req.ValueNumber,
		ValueDurationMs: req.ValueDurationMs,
		ValueText:       req.ValueText,
		Note:            req.Note,
	})
	if err != nil {
		if err.Error() == "INVALID_VALUE_TYPE" {
			Fail(c, http.StatusBadRequest, CodeInvalidValueType, "value does not match list type: "+list.ValueType)
			return
		}
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, entry)
}

func (h *EntryHandler) DeleteMyEntry(c *gin.Context) {
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

	if err := h.entries.DeleteOwnEntry(c.Request.Context(), listID, userID); err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "entry deleted"})
}

func (h *EntryHandler) DeleteEntry(c *gin.Context) {
	entryID, ok := middleware.ParseUUID(c.Param("entryId"))
	if !ok {
		ValidationError(c, "invalid entry id")
		return
	}

	if err := h.entries.DeleteEntry(c.Request.Context(), entryID); err != nil {
		InternalError(c)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "entry deleted"})
}
