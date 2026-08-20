package product

import (
	"context"
	"io"
	"time"
)

type Visibility string

const (
	Private  Visibility = "private"
	Unlisted Visibility = "unlisted"
	Public   Visibility = "public"
)

type User struct {
	ID, Email, DisplayName string
	CreatedAt              time.Time
}

type Artwork struct {
	ID, UserID, ShareID                             string
	Title, Subtitle, Palette, EngineVersion         string
	Visibility                                      Visibility
	SVGKey, PNGKey, FingerprintKey, RecipeKey       string
	FeaturesJSON, EventsJSON, RecipeJSON, ScoreJSON []byte
	CreatedAt                                       time.Time
}

type Store interface {
	EnsureUser(context.Context, string, string) (User, error)
	CreateMagicToken(context.Context, string, string, time.Time) error
	ConsumeMagicToken(context.Context, string) (User, error)
	CreateSession(context.Context, string, string, time.Time) error
	UserForSession(context.Context, string) (User, error)
	RevokeSession(context.Context, string) error
	RevokeUserSessions(context.Context, string) error
	DeleteUser(context.Context, string) ([]Artwork, error)
	AllowRateLimit(context.Context, string, int, time.Duration) (bool, time.Duration, error)
	StartGeneration(context.Context, string, string, int) error
	FinishGeneration(context.Context, string, string, string, string) error
	CreateArtwork(context.Context, Artwork) error
	ListArtworks(context.Context, string) ([]Artwork, error)
	ListPublicArtworks(context.Context, int) ([]Artwork, error)
	GetArtwork(context.Context, string) (Artwork, error)
	GetArtworkByShareID(context.Context, string) (Artwork, error)
	SetVisibility(context.Context, string, string, Visibility) error
	RotateShareID(context.Context, string, string, string) error
	DeleteArtwork(context.Context, string, string) (Artwork, error)
}

type Objects interface {
	Put(context.Context, string, string, []byte) error
	Get(context.Context, string) ([]byte, string, error)
	Delete(context.Context, string) error
}

type ArtifactWriter interface {
	Write(ctx context.Context, key, contentType string, source io.Reader) error
}
