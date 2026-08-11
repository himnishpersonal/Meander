package product

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("record not found")

type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]User
	byEmail  map[string]string
	artworks map[string]Artwork
	byShare  map[string]string
	magic    map[string]memoryToken
	sessions map[string]memoryToken
}

type memoryToken struct {
	UserID    string
	ExpiresAt time.Time
	Used      bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{users: map[string]User{}, byEmail: map[string]string{}, artworks: map[string]Artwork{}, byShare: map[string]string{}, magic: map[string]memoryToken{}, sessions: map[string]memoryToken{}}
}

func (s *MemoryStore) CreateMagicToken(_ context.Context, tokenHash, userID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.magic[tokenHash] = memoryToken{UserID: userID, ExpiresAt: expiresAt}
	return nil
}
func (s *MemoryStore) ConsumeMagicToken(_ context.Context, tokenHash string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.magic[tokenHash]
	if !ok || token.Used || time.Now().After(token.ExpiresAt) {
		return User{}, ErrNotFound
	}
	token.Used = true
	s.magic[tokenHash] = token
	user, ok := s.users[token.UserID]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}
func (s *MemoryStore) CreateSession(_ context.Context, tokenHash, userID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[tokenHash] = memoryToken{UserID: userID, ExpiresAt: expiresAt}
	return nil
}
func (s *MemoryStore) UserForSession(_ context.Context, tokenHash string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.sessions[tokenHash]
	if !ok || token.Used || time.Now().After(token.ExpiresAt) {
		return User{}, ErrNotFound
	}
	user, ok := s.users[token.UserID]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}
func (s *MemoryStore) RevokeSession(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.sessions[tokenHash]
	if !ok {
		return ErrNotFound
	}
	token.Used = true
	s.sessions[tokenHash] = token
	return nil
}

func (s *MemoryStore) EnsureUser(_ context.Context, email, displayName string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id := s.byEmail[email]; id != "" {
		return s.users[id], nil
	}
	u := User{ID: "local-" + stableID(email), Email: email, DisplayName: displayName, CreatedAt: time.Now().UTC()}
	s.users[u.ID], s.byEmail[email] = u, u.ID
	return u, nil
}

func (s *MemoryStore) CreateArtwork(_ context.Context, a Artwork) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[a.UserID]; !ok {
		return ErrNotFound
	}
	s.artworks[a.ID], s.byShare[a.ShareID] = a, a.ID
	return nil
}

func (s *MemoryStore) ListArtworks(_ context.Context, userID string) ([]Artwork, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Artwork, 0)
	for _, a := range s.artworks {
		if a.UserID == userID {
			items = append(items, a)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) GetArtwork(_ context.Context, id string) (Artwork, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.artworks[id]
	if !ok {
		return Artwork{}, ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) GetArtworkByShareID(ctx context.Context, shareID string) (Artwork, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id := s.byShare[shareID]
	a, ok := s.artworks[id]
	if !ok {
		return Artwork{}, ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) SetVisibility(_ context.Context, id, userID string, visibility Visibility) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.artworks[id]
	if !ok || a.UserID != userID {
		return ErrNotFound
	}
	a.Visibility = visibility
	s.artworks[id] = a
	return nil
}

func (s *MemoryStore) DeleteArtwork(_ context.Context, id, userID string) (Artwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.artworks[id]
	if !ok || a.UserID != userID {
		return Artwork{}, ErrNotFound
	}
	delete(s.artworks, id)
	delete(s.byShare, a.ShareID)
	return a, nil
}
