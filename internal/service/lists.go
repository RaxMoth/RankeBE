package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "ranke-be/internal/db/sqlc"
)

type ListService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewListService(queries *db.Queries, pool *pgxpool.Pool) *ListService {
	return &ListService{queries: queries, pool: pool}
}

// CreateListInput collects the create-list form. Optional fields are
// pgtype-wrapped so a NULL `description` (etc.) is preserved.
type CreateListInput struct {
	OwnerID      pgtype.UUID
	Title        string
	Description  pgtype.Text
	ValueType    string
	RankOrder    string
	IsPublic     bool
	Category     pgtype.Text
	TelegramLink pgtype.Text
	WhatsappLink pgtype.Text
	DiscordLink  pgtype.Text
}

// UpdateListInput uses pgtype.* for every field so the caller can express
// "leave this column alone" with an invalid pgtype value (the SQL uses
// COALESCE(narg, column) to keep the existing value).
type UpdateListInput struct {
	ID           pgtype.UUID
	Title        pgtype.Text
	Description  pgtype.Text
	IsPublic     pgtype.Bool
	Locked       pgtype.Bool
	Category     pgtype.Text
	TelegramLink pgtype.Text
	WhatsappLink pgtype.Text
	DiscordLink  pgtype.Text
}

// RankUpdate is one row in a bulk-rank PATCH.
type RankUpdate struct {
	EntryID pgtype.UUID
	Rank    int
}

func (s *ListService) CreateList(ctx context.Context, in CreateListInput) (*db.List, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	list, err := qtx.CreateList(ctx, db.CreateListParams{
		OwnerID:      in.OwnerID,
		Title:        in.Title,
		Description:  in.Description,
		ValueType:    in.ValueType,
		RankOrder:    in.RankOrder,
		IsPublic:     in.IsPublic,
		Category:     in.Category,
		TelegramLink: in.TelegramLink,
		WhatsappLink: in.WhatsappLink,
		DiscordLink:  in.DiscordLink,
	})
	if err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}

	if _, err := qtx.CreateListMember(ctx, db.CreateListMemberParams{
		ListID: list.ID,
		UserID: in.OwnerID,
		Role:   "owner",
	}); err != nil {
		return nil, fmt.Errorf("create owner member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &list, nil
}

func (s *ListService) GetUserLists(ctx context.Context, userID pgtype.UUID) ([]db.GetUserListsRow, error) {
	return s.queries.GetUserLists(ctx, userID)
}

func (s *ListService) GetListByID(ctx context.Context, listID pgtype.UUID) (*db.List, error) {
	list, err := s.queries.GetListByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	return &list, nil
}

func (s *ListService) GetMemberRole(ctx context.Context, listID, userID pgtype.UUID) (string, error) {
	member, err := s.queries.GetListMember(ctx, db.GetListMemberParams{ListID: listID, UserID: userID})
	if err != nil {
		return "", err
	}
	return member.Role, nil
}

// GetMemberCount returns the number of members of a list. Used to populate
// `memberCount` in the list DTO.
func (s *ListService) GetMemberCount(ctx context.Context, listID pgtype.UUID) (int, error) {
	rows, err := s.queries.ListMembers(ctx, listID)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *ListService) UpdateList(ctx context.Context, in UpdateListInput) (*db.List, error) {
	list, err := s.queries.UpdateList(ctx, db.UpdateListParams{
		ID:           in.ID,
		Title:        in.Title,
		Description:  in.Description,
		IsPublic:     in.IsPublic,
		Locked:       in.Locked,
		Category:     in.Category,
		TelegramLink: in.TelegramLink,
		WhatsappLink: in.WhatsappLink,
		DiscordLink:  in.DiscordLink,
	})
	if err != nil {
		return nil, fmt.Errorf("update list: %w", err)
	}
	return &list, nil
}

func (s *ListService) DeleteList(ctx context.Context, listID pgtype.UUID) error {
	return s.queries.DeleteList(ctx, listID)
}

func (s *ListService) JoinPublicList(ctx context.Context, listID, userID pgtype.UUID) error {
	list, err := s.queries.GetListByID(ctx, listID)
	if err != nil {
		return fmt.Errorf("list not found")
	}
	if !list.IsPublic {
		return errors.New("list is private, use an invite link")
	}

	// Idempotent — if already a member, return success
	if _, err := s.queries.GetListMember(ctx, db.GetListMemberParams{ListID: listID, UserID: userID}); err == nil {
		return nil
	}

	_, err = s.queries.CreateListMember(ctx, db.CreateListMemberParams{
		ListID: listID,
		UserID: userID,
		Role:   "member",
	})
	return err
}

func (s *ListService) GetInvitePreview(ctx context.Context, token pgtype.UUID) (*db.GetInvitePreviewRow, error) {
	row, err := s.queries.GetInvitePreview(ctx, token)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ListService) JoinByInvite(ctx context.Context, inviteToken pgtype.UUID, userID pgtype.UUID) (*db.List, error) {
	list, err := s.queries.GetListByInviteToken(ctx, inviteToken)
	if err != nil {
		return nil, fmt.Errorf("invalid invite token")
	}

	if _, err := s.queries.GetListMember(ctx, db.GetListMemberParams{ListID: list.ID, UserID: userID}); err == nil {
		return &list, nil
	}

	if _, err := s.queries.CreateListMember(ctx, db.CreateListMemberParams{
		ListID: list.ID,
		UserID: userID,
		Role:   "member",
	}); err != nil {
		return nil, fmt.Errorf("join list: %w", err)
	}

	return &list, nil
}

// RegenerateInviteToken rotates the invite token (invalidates any old links).
func (s *ListService) RegenerateInviteToken(ctx context.Context, listID pgtype.UUID) (pgtype.UUID, error) {
	return s.queries.RegenerateInviteToken(ctx, listID)
}

// PublicListCursor is the keyset position for public-list pagination: the
// (updated_at, id) of the last row of the previous page. A nil cursor starts
// from the top of the (updated_at DESC, id DESC) ordering.
type PublicListCursor struct {
	UpdatedAt time.Time
	ID        pgtype.UUID
}

// SearchPublicLists searches public lists by title/description and category,
// returning at most `limit` rows past `cursor`. `query`/`category` are optional
// (empty strings drop the filter); `cursor` is nil for the first page.
func (s *ListService) SearchPublicLists(ctx context.Context, query, category string, limit int32, cursor *PublicListCursor) ([]db.SearchPublicListsRow, error) {
	params := db.SearchPublicListsParams{Lim: limit}
	if query != "" {
		params.Q = pgtype.Text{String: query, Valid: true}
	}
	if category != "" {
		params.Category = pgtype.Text{String: category, Valid: true}
	}
	if cursor != nil {
		params.CursorUpdatedAt = pgtype.Timestamptz{Time: cursor.UpdatedAt, Valid: true}
		params.CursorID = cursor.ID
	}
	return s.queries.SearchPublicLists(ctx, params)
}

func (s *ListService) GetMembers(ctx context.Context, listID pgtype.UUID) ([]db.ListMembersRow, error) {
	return s.queries.ListMembers(ctx, listID)
}

func (s *ListService) UpdateMemberRole(ctx context.Context, listID, targetUserID pgtype.UUID, role string) (*db.ListMember, error) {
	list, err := s.queries.GetListByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if targetUserID == list.OwnerID {
		return nil, errors.New("cannot change the owner's role")
	}

	member, err := s.queries.UpdateMemberRole(ctx, db.UpdateMemberRoleParams{
		ListID: listID,
		UserID: targetUserID,
		Role:   role,
	})
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (s *ListService) RemoveMember(ctx context.Context, listID, targetUserID pgtype.UUID) error {
	list, err := s.queries.GetListByID(ctx, listID)
	if err != nil {
		return err
	}
	if targetUserID == list.OwnerID {
		return errors.New("cannot remove the list owner")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	if err := qtx.DeleteListMember(ctx, db.DeleteListMemberParams{ListID: listID, UserID: targetUserID}); err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	if err := qtx.DeleteEntryByListAndUser(ctx, db.DeleteEntryByListAndUserParams{ListID: listID, UserID: targetUserID}); err != nil {
		return fmt.Errorf("delete entry: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *ListService) BulkUpdateRanks(ctx context.Context, updates []RankUpdate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)
	for _, u := range updates {
		if err := qtx.UpdateEntryManualRank(ctx, db.UpdateEntryManualRankParams{
			ID:         u.EntryID,
			ManualRank: pgtype.Int4{Int32: int32(u.Rank), Valid: true},
		}); err != nil {
			return fmt.Errorf("update rank for entry: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// GetRankedEntries returns ranked entries for a list, dispatching on the
// list's value_type / rank_order. Returns one of five concrete row slice
// types — callers should run it through dto.MapRankedEntries.
func (s *ListService) GetRankedEntries(ctx context.Context, list *db.List) (any, error) {
	switch list.ValueType {
	case "number":
		if list.RankOrder == "desc" {
			return s.queries.GetRankedEntriesByNumberDesc(ctx, list.ID)
		}
		return s.queries.GetRankedEntriesByNumber(ctx, list.ID)
	case "duration":
		if list.RankOrder == "desc" {
			return s.queries.GetRankedEntriesByDurationDesc(ctx, list.ID)
		}
		return s.queries.GetRankedEntriesByDuration(ctx, list.ID)
	case "text":
		return s.queries.GetRankedEntriesByText(ctx, list.ID)
	default:
		return nil, fmt.Errorf("unknown value type: %s", list.ValueType)
	}
}

// NOTE: a previous version of this service had GetUserCurrentRank +
// GetUserRankFromRow helpers used to look up one user's rank for the
// home-feed `ownRank`. They were removed once GetUserLists started
// computing own_rank inline as part of the same query (saves N+1
// round-trips). The five GetCurrentRankBy* queries still exist and are
// used by EntryService.UpsertEntry to capture previous_rank.
