package service

import (
	"context"
	"errors"
	"fmt"

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

func (s *ListService) CreateList(ctx context.Context, ownerID pgtype.UUID, title string, description pgtype.Text, valueType, rankOrder string, isPublic bool) (*db.List, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	list, err := qtx.CreateList(ctx, db.CreateListParams{
		OwnerID:     ownerID,
		Title:       title,
		Description: description,
		ValueType:   valueType,
		RankOrder:   rankOrder,
		IsPublic:    isPublic,
	})
	if err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}

	_, err = qtx.CreateListMember(ctx, db.CreateListMemberParams{
		ListID: list.ID,
		UserID: ownerID,
		Role:   "owner",
	})
	if err != nil {
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

func (s *ListService) UpdateList(ctx context.Context, listID pgtype.UUID, title string, description pgtype.Text, isPublic bool) (*db.List, error) {
	list, err := s.queries.UpdateList(ctx, db.UpdateListParams{
		ID:          listID,
		Title:       title,
		Description: description,
		IsPublic:    isPublic,
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
	_, err = s.queries.GetListMember(ctx, db.GetListMemberParams{ListID: listID, UserID: userID})
	if err == nil {
		return nil // Already a member
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

	// Idempotent
	_, err = s.queries.GetListMember(ctx, db.GetListMemberParams{ListID: list.ID, UserID: userID})
	if err == nil {
		return &list, nil
	}

	_, err = s.queries.CreateListMember(ctx, db.CreateListMemberParams{
		ListID: list.ID,
		UserID: userID,
		Role:   "member",
	})
	if err != nil {
		return nil, fmt.Errorf("join list: %w", err)
	}

	return &list, nil
}

func (s *ListService) GetMembers(ctx context.Context, listID pgtype.UUID) ([]db.ListMembersRow, error) {
	return s.queries.ListMembers(ctx, listID)
}

func (s *ListService) UpdateMemberRole(ctx context.Context, listID, targetUserID pgtype.UUID, role string) (*db.ListMember, error) {
	// Check the target is not the owner
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

type RankUpdate struct {
	EntryID pgtype.UUID
	Rank    int
}

// GetRankedEntries returns ranked entries for a list, using the appropriate query based on value_type and rank_order.
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
