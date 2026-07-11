package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ranke-be/internal/handler/dto"
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

// upsertEntryRequest accepts camelCase to match the mobile client.
// All fields are optional — the value-type validation happens server-side
// based on the list's value_type column.
type upsertEntryRequest struct {
	ValueNumber     *float64 `json:"valueNumber"`
	ValueDurationMs *int64   `json:"valueDurationMs"`
	ValueText       *string  `json:"valueText"`
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

	list, err := h.lists.GetListByID(c.Request.Context(), listID)
	if err != nil {
		Fail(c, http.StatusNotFound, CodeListNotFound, "list not found")
		return
	}

	entry, err := h.entries.UpsertEntry(c.Request.Context(), listID, userID, list, service.UpsertEntryInput{
		ValueNumber:     req.ValueNumber,
		ValueDurationMs: req.ValueDurationMs,
		ValueText:       req.ValueText,
		Note:            req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidValueType):
			Fail(c, http.StatusBadRequest, CodeInvalidValueType, "value does not match list type: "+list.ValueType)
		case errors.Is(err, service.ErrListLocked):
			Fail(c, http.StatusForbidden, CodeForbidden, "this list is locked")
		default:
			InternalErrorLog(c, "upsertEntry", err)
		}
		return
	}

	Success(c, http.StatusOK, dto.MapEntry(*entry))
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
		InternalErrorLog(c, "deleteMyEntry", err)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "entry deleted"})
}

// DeleteEntry — owner/admin only (delete any user's entry).
func (h *EntryHandler) DeleteEntry(c *gin.Context) {
	entryID, ok := middleware.ParseUUID(c.Param("entryId"))
	if !ok {
		ValidationError(c, "invalid entry id")
		return
	}

	if err := h.entries.DeleteEntry(c.Request.Context(), entryID); err != nil {
		InternalErrorLog(c, "deleteEntry", err)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "entry deleted"})
}

// ── moderation (owner/admin) ─────────────────────────────────────────

func (h *EntryHandler) GetPendingEntries(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	rows, err := h.entries.GetPendingEntries(c.Request.Context(), listID)
	if err != nil {
		InternalErrorLog(c, "getPendingEntries", err)
		return
	}

	out := make([]dto.RankedEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.MapPendingEntryRow(r))
	}
	Success(c, http.StatusOK, out)
}

func (h *EntryHandler) ApproveEntry(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}
	entryID, ok := middleware.ParseUUID(c.Param("entryId"))
	if !ok {
		ValidationError(c, "invalid entry id")
		return
	}

	if err := h.entries.ApproveEntry(c.Request.Context(), listID, entryID); err != nil {
		InternalErrorLog(c, "approveEntry", err)
		return
	}
	Success(c, http.StatusOK, gin.H{"message": "approved"})
}

func (h *EntryHandler) RejectEntry(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}
	entryID, ok := middleware.ParseUUID(c.Param("entryId"))
	if !ok {
		ValidationError(c, "invalid entry id")
		return
	}

	if err := h.entries.RejectEntry(c.Request.Context(), listID, entryID); err != nil {
		InternalErrorLog(c, "rejectEntry", err)
		return
	}
	Success(c, http.StatusOK, gin.H{"message": "rejected"})
}
