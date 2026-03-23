package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	platformpostgres "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/postgres"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

type GameRepository struct {
	pool *pgxpool.Pool
}

func NewGameRepository(client *platformpostgres.Client) *GameRepository {
	if client == nil || !client.Enabled() {
		return nil
	}

	return &GameRepository{
		pool: client.Pool(),
	}
}

func (r *GameRepository) UpsertBySteamAppID(ctx context.Context, input model.GameUpsertInput) (*model.Game, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("game repository is not configured")
	}

	var (
		game        model.Game
		coverURL    sql.NullString
		genre       sql.NullString
		releaseYear sql.NullInt32
	)

	query := `
		insert into public.games (steam_app_id, title, cover_url, genre, release_year)
		values ($1, $2, $3, $4, $5)
		on conflict (steam_app_id) do update
		set
			title = case
				when $6 then public.games.title
				else excluded.title
			end,
			cover_url = coalesce(excluded.cover_url, public.games.cover_url),
			genre = coalesce(excluded.genre, public.games.genre),
			release_year = coalesce(excluded.release_year, public.games.release_year)
		returning id, steam_app_id, title, cover_url, genre, release_year
	`

	if err := r.pool.QueryRow(
		ctx,
		query,
		input.SteamAppID,
		input.Title,
		input.CoverURL,
		input.Genre,
		input.ReleaseYear,
		input.PreferExistingTitle,
	).Scan(
		&game.ID,
		&game.SteamAppID,
		&game.Title,
		&coverURL,
		&genre,
		&releaseYear,
	); err != nil {
		return nil, fmt.Errorf("upsert game by steam_app_id: %w", err)
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
