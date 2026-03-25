package mysql

import (
	"context"
	"database/sql"
	"fmt"

	platformmysql "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/mysql"
	platformuuid "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/uuid"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type GameRepository struct {
	db *sql.DB
}

func NewGameRepository(client *platformmysql.Client) *GameRepository {
	if client == nil || !client.Enabled() {
		return nil
	}

	return &GameRepository{
		db: client.DB(),
	}
}

func (r *GameRepository) UpsertBySteamAppID(ctx context.Context, input model.GameUpsertInput) (*model.Game, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("game repository is not configured")
	}

	gameID, err := platformuuid.NewString()
	if err != nil {
		return nil, fmt.Errorf("generate game id: %w", err)
	}

	query := `
		insert into games (id, steam_app_id, title, cover_url, genre, release_year)
		values (?, ?, ?, ?, ?, ?)
		on duplicate key update
			title = if(?, games.title, values(title)),
			cover_url = coalesce(values(cover_url), games.cover_url),
			genre = coalesce(values(genre), games.genre),
			release_year = coalesce(values(release_year), games.release_year)
	`

	if _, err := r.db.ExecContext(
		ctx,
		query,
		gameID,
		input.SteamAppID,
		input.Title,
		input.CoverURL,
		input.Genre,
		input.ReleaseYear,
		input.PreferExistingTitle,
	); err != nil {
		return nil, fmt.Errorf("upsert game by steam_app_id: %w", err)
	}

	return r.findBySteamAppID(ctx, input.SteamAppID)
}

func (r *GameRepository) findBySteamAppID(ctx context.Context, appID string) (*model.Game, error) {
	var (
		game        model.Game
		coverURL    sql.NullString
		genre       sql.NullString
		releaseYear sql.NullInt32
	)

	query := `
		select id, steam_app_id, title, cover_url, genre, release_year
		from games
		where steam_app_id = ?
		limit 1
	`

	if err := r.db.QueryRowContext(ctx, query, appID).Scan(
		&game.ID,
		&game.SteamAppID,
		&game.Title,
		&coverURL,
		&genre,
		&releaseYear,
	); err != nil {
		return nil, fmt.Errorf("load game by steam_app_id: %w", err)
	}

	if coverURL.Valid {
		game.CoverURL = coverURL.String
	}
	if genre.Valid {
		game.Genre = genre.String
	}
	if releaseYear.Valid {
		year := int(releaseYear.Int32)
		game.ReleaseYear = &year
	}

	return &game, nil
}
