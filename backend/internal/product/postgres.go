package product

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(ctx context.Context, url string) (*PostgresStore, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err = p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &PostgresStore{pool: p}, nil
}
func (s *PostgresStore) Close() { s.pool.Close() }

func (s *PostgresStore) EnsureUser(ctx context.Context, email, displayName string) (User, error) {
	const q = `INSERT INTO users (id,email,display_name) VALUES ($1,$2,$3)
		ON CONFLICT (email) DO UPDATE SET display_name = COALESCE(NULLIF(EXCLUDED.display_name,''), users.display_name)
		RETURNING id,email,COALESCE(display_name,''),created_at`
	var u User
	err := s.pool.QueryRow(ctx, q, NewID("usr_"), email, displayName).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	return u, err
}
func (s *PostgresStore) CreateMagicToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO magic_link_tokens (token_hash,user_id,expires_at) VALUES ($1,$2,$3)`, tokenHash, userID, expiresAt)
	return err
}
func (s *PostgresStore) ConsumeMagicToken(ctx context.Context, tokenHash string) (User, error) {
	const q = `UPDATE magic_link_tokens SET used_at=now() WHERE token_hash=$1 AND used_at IS NULL AND expires_at > now() RETURNING user_id`
	var userID string
	if err := s.pool.QueryRow(ctx, q, tokenHash).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	} else if err != nil {
		return User{}, err
	}
	var u User
	err := s.pool.QueryRow(ctx, `SELECT id,email,COALESCE(display_name,''),created_at FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
func (s *PostgresStore) CreateSession(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO sessions (id_hash,user_id,expires_at) VALUES ($1,$2,$3)`, tokenHash, userID, expiresAt)
	return err
}
func (s *PostgresStore) UserForSession(ctx context.Context, tokenHash string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT u.id,u.email,COALESCE(u.display_name,''),u.created_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now() AND u.deleted_at IS NULL`, tokenHash).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
func (s *PostgresStore) RevokeSession(ctx context.Context, tokenHash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=now() WHERE id_hash=$1 AND revoked_at IS NULL`, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) CreateArtwork(ctx context.Context, a Artwork) error {
	const q = `INSERT INTO artworks (id,user_id,share_id,title,subtitle,palette,engine_version,visibility,svg_key,png_key,fingerprint_key,recipe_key,features,events,recipe,score,created_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,$15::jsonb,$16::jsonb,$17)`
	_, err := s.pool.Exec(ctx, q, a.ID, a.UserID, a.ShareID, a.Title, a.Subtitle, a.Palette, a.EngineVersion, a.Visibility, a.SVGKey, a.PNGKey, a.FingerprintKey, a.RecipeKey, a.FeaturesJSON, a.EventsJSON, a.RecipeJSON, a.ScoreJSON, a.CreatedAt)
	return err
}

const artworkColumns = `id,user_id,share_id,title,subtitle,palette,engine_version,visibility,svg_key,png_key,fingerprint_key,recipe_key,features::text,events::text,recipe::text,score::text,created_at`

func scanArtwork(row pgx.Row) (Artwork, error) {
	var a Artwork
	err := row.Scan(&a.ID, &a.UserID, &a.ShareID, &a.Title, &a.Subtitle, &a.Palette, &a.EngineVersion, &a.Visibility, &a.SVGKey, &a.PNGKey, &a.FingerprintKey, &a.RecipeKey, &a.FeaturesJSON, &a.EventsJSON, &a.RecipeJSON, &a.ScoreJSON, &a.CreatedAt)
	return a, err
}
func (s *PostgresStore) ListArtworks(ctx context.Context, userID string) ([]Artwork, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+artworkColumns+` FROM artworks WHERE user_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Artwork{}
	for rows.Next() {
		a, err := scanArtwork(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
func (s *PostgresStore) ListPublicArtworks(ctx context.Context, limit int) ([]Artwork, error) {
	if limit <= 0 || limit > 100 {
		limit = 48
	}
	rows, err := s.pool.Query(ctx, `SELECT `+artworkColumns+` FROM artworks WHERE visibility='public' AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Artwork{}
	for rows.Next() {
		a, err := scanArtwork(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
func (s *PostgresStore) GetArtwork(ctx context.Context, id string) (Artwork, error) {
	return s.get(ctx, `SELECT `+artworkColumns+` FROM artworks WHERE id=$1 AND deleted_at IS NULL`, id)
}
func (s *PostgresStore) GetArtworkByShareID(ctx context.Context, shareID string) (Artwork, error) {
	return s.get(ctx, `SELECT `+artworkColumns+` FROM artworks WHERE share_id=$1 AND deleted_at IS NULL`, shareID)
}
func (s *PostgresStore) get(ctx context.Context, query, value string) (Artwork, error) {
	a, err := scanArtwork(s.pool.QueryRow(ctx, query, value))
	if errors.Is(err, pgx.ErrNoRows) {
		return Artwork{}, ErrNotFound
	}
	return a, err
}
func (s *PostgresStore) SetVisibility(ctx context.Context, id, userID string, visibility Visibility) error {
	tag, err := s.pool.Exec(ctx, `UPDATE artworks SET visibility=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`, visibility, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
func (s *PostgresStore) DeleteArtwork(ctx context.Context, id, userID string) (Artwork, error) {
	a, err := s.GetArtwork(ctx, id)
	if err != nil || a.UserID != userID {
		return Artwork{}, ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `UPDATE artworks SET deleted_at=$1 WHERE id=$2`, time.Now().UTC(), id)
	return a, err
}
