package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	db "ranke-be/internal/db/sqlc"
	"ranke-be/internal/handler/dto"
	"ranke-be/internal/middleware"
	"ranke-be/internal/service"
)

type ListHandler struct {
	lists *service.ListService
}

func NewListHandler(lists *service.ListService) *ListHandler {
	return &ListHandler{lists: lists}
}

// ── request DTOs (camelCase) ─────────────────────────────────────────

type createListRequest struct {
	Title        string  `json:"title"        binding:"required,max=255"`
	Description  *string `json:"description"  binding:"omitempty,max=2000"`
	ValueType    string  `json:"valueType"    binding:"required,oneof=number duration text"`
	RankOrder    string  `json:"rankOrder"    binding:"required,oneof=asc desc"`
	IsPublic     *bool   `json:"isPublic"`
	Category     *string `json:"category"`
	TelegramLink *string `json:"telegramLink"`
	WhatsappLink *string `json:"whatsappLink"`
	DiscordLink  *string `json:"discordLink"`
}

type updateListRequest struct {
	Title        *string `json:"title"        binding:"omitempty,max=255"`
	Description  *string `json:"description"  binding:"omitempty,max=2000"`
	IsPublic     *bool   `json:"isPublic"`
	Locked       *bool   `json:"locked"`
	Category     *string `json:"category"`
	TelegramLink *string `json:"telegramLink"`
	WhatsappLink *string `json:"whatsappLink"`
	DiscordLink  *string `json:"discordLink"`
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}

type rankUpdateItem struct {
	EntryID string `json:"entryId" binding:"required"`
	Rank    int    `json:"rank"    binding:"required"`
}

// ── helpers ──────────────────────────────────────────────────────────

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func toPgBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

// fetchListWithEntries assembles the full RankedList DTO for /lists/:id
// (and the create/update endpoints) — pulling member count, the caller's
// role, and the ranked entries.
func (h *ListHandler) fetchListWithEntries(c *gin.Context, list *db.List, userID pgtype.UUID) (dto.RankedList, error) {
	memberCount, err := h.lists.GetMemberCount(c.Request.Context(), list.ID)
	if err != nil {
		return dto.RankedList{}, err
	}

	var rolePtr *string
	if role, err := h.lists.GetMemberRole(c.Request.Context(), list.ID, userID); err == nil {
		rolePtr = &role
	}

	rankedRows, err := h.lists.GetRankedEntries(c.Request.Context(), list)
	if err != nil {
		return dto.RankedList{}, err
	}
	entries := dto.MapRankedEntries(rankedRows)

	return dto.MapList(*list, memberCount, rolePtr, entries), nil
}

// ── handlers ─────────────────────────────────────────────────────────

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

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	list, err := h.lists.CreateList(c.Request.Context(), service.CreateListInput{
		OwnerID:      userID,
		Title:        req.Title,
		Description:  toPgText(req.Description),
		ValueType:    req.ValueType,
		RankOrder:    req.RankOrder,
		IsPublic:     isPublic,
		Category:     toPgText(req.Category),
		TelegramLink: toPgText(req.TelegramLink),
		WhatsappLink: toPgText(req.WhatsappLink),
		DiscordLink:  toPgText(req.DiscordLink),
	})
	if err != nil {
		InternalErrorLog(c, "createList", err)
		return
	}

	resp, err := h.fetchListWithEntries(c, list, userID)
	if err != nil {
		InternalErrorLog(c, "createList.fetch", err)
		return
	}
	Success(c, http.StatusCreated, resp)
}

func (h *ListHandler) GetUserLists(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	rows, err := h.lists.GetUserLists(c.Request.Context(), userID)
	if err != nil {
		InternalErrorLog(c, "getUserLists", err)
		return
	}

	// Single round-trip: own_rank is computed inline by the query. 0
	// is the "no rank" sentinel (see queries/lists.sql) — convert to
	// a nil pointer before the DTO mapper to keep the wire contract
	// consistent (omitempty drops it from the JSON).
	out := make([]dto.ListSummary, 0, len(rows))
	for _, r := range rows {
		var ownRank *int
		if r.OwnRank > 0 {
			v := int(r.OwnRank)
			ownRank = &v
		}
		out = append(out, dto.MapUserListsRow(r, ownRank))
	}

	Success(c, http.StatusOK, out)
}

// Public-list pagination knobs. The mobile discover screen pages through
// results 30 at a time; the cap keeps a hostile `?limit=` from asking for
// the whole table in one request.
const (
	publicListsDefaultLimit = 30
	publicListsMaxLimit     = 100
)

func (h *ListHandler) SearchPublic(c *gin.Context) {
	q := c.Query("q")
	cat := c.Query("category")
	limit := parsePublicListsLimit(c.Query("limit"))

	cursor, ok := decodePublicListCursor(c.Query("cursor"))
	if !ok {
		ValidationError(c, "invalid cursor")
		return
	}

	// Fetch limit+1 so we can tell whether a further page exists without a
	// second COUNT query: if we get the extra row, there's more, and we trim
	// it off before returning.
	rows, err := h.lists.SearchPublicLists(c.Request.Context(), q, cat, int32(limit)+1, cursor)
	if err != nil {
		InternalErrorLog(c, "searchPublic", err)
		return
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	out := make([]dto.ListSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.MapPublicListsRow(r))
	}

	// The next cursor rides in a response header so the body stays a bare
	// array — the mobile client unwraps `data` as a list and would break on
	// an envelope change. Absent header == no further pages.
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c.Header("X-Next-Cursor", encodePublicListCursor(last.UpdatedAt, last.ID))
	}

	Success(c, http.StatusOK, out)
}

// parsePublicListsLimit clamps a caller-supplied `?limit=` into
// [1, publicListsMaxLimit], falling back to the default on empty/garbage input.
func parsePublicListsLimit(raw string) int {
	if raw == "" {
		return publicListsDefaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return publicListsDefaultLimit
	}
	if n > publicListsMaxLimit {
		return publicListsMaxLimit
	}
	return n
}

// encodePublicListCursor packs the keyset position (updated_at, id) into an
// opaque base64url token: `<unixMicros>:<uuid>`.
func encodePublicListCursor(updatedAt pgtype.Timestamptz, id pgtype.UUID) string {
	raw := fmt.Sprintf("%d:%s", updatedAt.Time.UnixMicro(), dto.FormatUUID(id))
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodePublicListCursor reverses encodePublicListCursor. An empty string
// yields (nil, true) — a valid "first page" request. Malformed input yields
// (nil, false) so the handler can 400 rather than silently ignore it.
func decodePublicListCursor(raw string) (*service.PublicListCursor, bool) {
	if raw == "" {
		return nil, true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, false
	}
	micros, idStr, found := strings.Cut(string(decoded), ":")
	if !found {
		return nil, false
	}
	us, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return nil, false
	}
	id, ok := middleware.ParseUUID(idStr)
	if !ok {
		return nil, false
	}
	return &service.PublicListCursor{UpdatedAt: time.UnixMicro(us).UTC(), ID: id}, true
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
		if _, err := h.lists.GetMemberRole(c.Request.Context(), listID, userID); err != nil {
			Forbidden(c)
			return
		}
	}

	resp, err := h.fetchListWithEntries(c, list, userID)
	if err != nil {
		InternalErrorLog(c, "getListByID.fetch", err)
		return
	}
	Success(c, http.StatusOK, resp)
}

func (h *ListHandler) Update(c *gin.Context) {
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

	var req updateListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, err.Error())
		return
	}

	list, err := h.lists.UpdateList(c.Request.Context(), service.UpdateListInput{
		ID:           listID,
		Title:        toPgText(req.Title),
		Description:  toPgText(req.Description),
		IsPublic:     toPgBool(req.IsPublic),
		Locked:       toPgBool(req.Locked),
		Category:     toPgText(req.Category),
		TelegramLink: toPgText(req.TelegramLink),
		WhatsappLink: toPgText(req.WhatsappLink),
		DiscordLink:  toPgText(req.DiscordLink),
	})
	if err != nil {
		InternalErrorLog(c, "updateList", err)
		return
	}

	resp, err := h.fetchListWithEntries(c, list, userID)
	if err != nil {
		InternalErrorLog(c, "updateList.fetch", err)
		return
	}
	Success(c, http.StatusOK, resp)
}

func (h *ListHandler) Delete(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	if err := h.lists.DeleteList(c.Request.Context(), listID); err != nil {
		InternalErrorLog(c, "deleteList", err)
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
	token, ok := middleware.ParseUUID(c.Param("token"))
	if !ok {
		ValidationError(c, "invalid invite token")
		return
	}

	preview, err := h.lists.GetInvitePreview(c.Request.Context(), token)
	if err != nil {
		Fail(c, http.StatusNotFound, CodeListNotFound, "invalid invite link")
		return
	}

	Success(c, http.StatusOK, dto.MapInvitePreviewRow(*preview))
}

func (h *ListHandler) JoinByInvite(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "authentication required")
		return
	}

	token, ok := middleware.ParseUUID(c.Param("token"))
	if !ok {
		ValidationError(c, "invalid invite token")
		return
	}

	list, err := h.lists.JoinByInvite(c.Request.Context(), token, userID)
	if err != nil {
		Fail(c, http.StatusBadRequest, CodeValidationError, err.Error())
		return
	}

	resp, err := h.fetchListWithEntries(c, list, userID)
	if err != nil {
		InternalErrorLog(c, "joinByInvite.fetch", err)
		return
	}
	Success(c, http.StatusOK, resp)
}

// GetInviteLink returns the current invite token (and a deep link) for a
// list. Owner/admin only — enforced by middleware.
func (h *ListHandler) GetInviteLink(c *gin.Context) {
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

	token := dto.FormatUUID(list.InviteToken)
	Success(c, http.StatusOK, dto.InviteLink{
		InviteLink:  "ranke://invite/" + token,
		InviteToken: token,
	})
}

// RegenerateInvite issues a fresh invite token (invalidates the old one).
// Owner/admin only.
func (h *ListHandler) RegenerateInvite(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	newToken, err := h.lists.RegenerateInviteToken(c.Request.Context(), listID)
	if err != nil {
		InternalErrorLog(c, "regenerateInvite", err)
		return
	}

	tokenStr := dto.FormatUUID(newToken)
	Success(c, http.StatusOK, dto.InviteLink{
		InviteLink:  "ranke://invite/" + tokenStr,
		InviteToken: tokenStr,
	})
}

func (h *ListHandler) GetMembers(c *gin.Context) {
	listID, ok := middleware.ParseUUID(c.Param("id"))
	if !ok {
		ValidationError(c, "invalid list id")
		return
	}

	rows, err := h.lists.GetMembers(c.Request.Context(), listID)
	if err != nil {
		InternalErrorLog(c, "getMembers", err)
		return
	}

	out := make([]dto.ListMember, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.MapMembersRow(r))
	}
	Success(c, http.StatusOK, out)
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

	if _, err := h.lists.UpdateMemberRole(c.Request.Context(), listID, targetUserID, req.Role); err != nil {
		Fail(c, http.StatusBadRequest, CodeValidationError, err.Error())
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "role updated"})
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
			ValidationError(c, "invalid entryId: "+item.EntryID)
			return
		}
		updates = append(updates, service.RankUpdate{
			EntryID: entryID,
			Rank:    item.Rank,
		})
	}

	if err := h.lists.BulkUpdateRanks(c.Request.Context(), updates); err != nil {
		InternalErrorLog(c, "bulkUpdateRanks", err)
		return
	}

	Success(c, http.StatusOK, gin.H{"message": "ranks updated"})
}
