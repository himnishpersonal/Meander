package product

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("record not found")
var ErrQuotaExceeded = errors.New("daily generation quota reached")

type MemoryStore struct {
	mu       sync.RWMutex
	users    map[string]User
	byEmail  map[string]string
	artworks map[string]Artwork
	byShare  map[string]string
	magic    map[string]memoryToken
	sessions map[string]memoryToken
	jobs     map[string]memoryJob
	rates    map[string]memoryRate
}

type memoryToken struct {
	UserID    string
	ExpiresAt time.Time
	Used      bool
}

type memoryJob struct {
	UserID, Status string
	CreatedAt      time.Time
}

type memoryRate struct {
	Count int
	Reset time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{users: map[string]User{}, byEmail: map[string]string{}, artworks: map[string]Artwork{}, byShare: map[string]string{}, magic: map[string]memoryToken{}, sessions: map[string]memoryToken{}, jobs: map[string]memoryJob{}, rates: map[string]memoryRate{}}
}

func (s *MemoryStore) StartGeneration(_ context.Context, id, userID string, dailyLimit int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff, count := time.Now().Add(-24*time.Hour), 0
	for _, job := range s.jobs {
		if job.UserID == userID && job.CreatedAt.After(cutoff) {
			count++
		}
	}
	if count >= dailyLimit {
		return ErrQuotaExceeded
	}
	s.jobs[id] = memoryJob{UserID: userID, Status: "processing", CreatedAt: time.Now().UTC()}
	return nil
}

func (s *MemoryStore) FinishGeneration(_ context.Context, id, status, _, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return ErrNotFound
	}
	job.Status = status
	s.jobs[id] = job
	return nil
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

func (s *MemoryStore) RevokeUserSessions(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, token := range s.sessions {
		if token.UserID == userID {
			token.Used = true
			s.sessions[key] = token
		}
	}
	return nil
}

func (s *MemoryStore) DeleteUser(_ context.Context, userID string) ([]Artwork, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return nil, ErrNotFound
	}
	artworks := make([]Artwork, 0)
	for id, artwork := range s.artworks {
		if artwork.UserID == userID {
			artworks = append(artworks, artwork)
			delete(s.artworks, id)
			delete(s.byShare, artwork.ShareID)
		}
	}
	for key, token := range s.sessions {
		if token.UserID == userID {
			delete(s.sessions, key)
		}
	}
	delete(s.byEmail, user.Email)
	delete(s.users, userID)
	return artworks, nil
}

func (s *MemoryStore) AllowRateLimit(_ context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		return true, 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	entry, ok := s.rates[key]
	if !ok || !now.Before(entry.Reset) {
		s.rates[key] = memoryRate{Count: 1, Reset: now.Add(window)}
		return true, 0, nil
	}
	if entry.Count >= limit {
		return false, time.Until(entry.Reset), nil
	}
	entry.Count++
	s.rates[key] = entry
	return true, 0, nil
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

func (s *MemoryStore) ListPublicArtworks(_ context.Context, limit int) ([]Artwork, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Artwork, 0)
	for _, a := range s.artworks {
		if a.Visibility == Public {
			items = append(items, a)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
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

func (s *MemoryStore) RotateShareID(_ context.Context, id, userID, shareID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.artworks[id]
	if !ok || a.UserID != userID {
		return ErrNotFound
	}
	delete(s.byShare, a.ShareID)
	a.ShareID = shareID
	s.artworks[id], s.byShare[shareID] = a, id
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
