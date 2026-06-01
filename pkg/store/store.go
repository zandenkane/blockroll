// Package store provides SQLite persistence for blockroll.
//
// All queries that return listings enforce the same visibility rules
// as pkg/privacy.Visible: the SQL WHERE clause mirrors the pure Go predicate.
package store

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// User represents a registered account.
type User struct {
	ID        string
	Email     string
	Password  string // bcrypt hash
	CreatedAt string
}

// Session ties a token to a user.
type Session struct {
	Token     string
	UserID    string
	CreatedAt string
}

// Neighborhood is a group of users sharing a join code.
type Neighborhood struct {
	ID        string
	Name      string
	JoinCode  string
	CreatedBy string
	CreatedAt string
}

// Listing is a skill or resource a user shares with their neighborhood.
type Listing struct {
	ID             string
	UserID         string
	NeighborhoodID string
	Title          string
	Description    string
	Category       string // skill, tool, food, other
	MinTier        int
	CreatedAt      string
	// Populated by joins, not stored in listings table directly.
	OwnerEmail string
}

// TrustEdge records the trust tier an owner assigns to a neighbor.
type TrustEdge struct {
	OwnerID    string
	NeighborID string
	Tier       int
}

// Store wraps a SQLite database connection.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at path and initializes the schema.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	// Enable WAL mode for better concurrency.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: enable foreign keys: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: init schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// newID generates a random hex ID (16 bytes = 32 hex chars).
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("store: rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// newJoinCode generates a 6 character alphanumeric code.
func newJoinCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I/1/O/0 to avoid confusion
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			panic(fmt.Sprintf("store: rand.Int failed: %v", err))
		}
		code[i] = chars[n.Int64()]
	}
	return string(code)
}

// --- Users ---

// CreateUser inserts a new user and returns the generated ID.
func (s *Store) CreateUser(email, passwordHash string) (string, error) {
	id := newID()
	_, err := s.db.Exec(
		"INSERT INTO users (id, email, password) VALUES (?, ?, ?)",
		id, email, passwordHash,
	)
	if err != nil {
		return "", fmt.Errorf("store: create user: %w", err)
	}
	return id, nil
}

// GetUserByEmail looks up a user by email. Returns nil if not found.
func (s *Store) GetUserByEmail(email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, email, password, created_at FROM users WHERE email = ?", email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID looks up a user by ID. Returns nil if not found.
func (s *Store) GetUserByID(id string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, email, password, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user by id: %w", err)
	}
	return u, nil
}

// --- Sessions ---

// CreateSession inserts a new session and returns the token.
func (s *Store) CreateSession(userID string) (string, error) {
	token := newID()
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, user_id) VALUES (?, ?)",
		token, userID,
	)
	if err != nil {
		return "", fmt.Errorf("store: create session: %w", err)
	}
	return token, nil
}

// GetSession returns the user ID for a session token. Returns "" if not found.
func (s *Store) GetSession(token string) (string, error) {
	var userID string
	err := s.db.QueryRow(
		"SELECT user_id FROM sessions WHERE token = ?", token,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: get session: %w", err)
	}
	return userID, nil
}

// DeleteSession removes a session by token.
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	return nil
}

// --- Neighborhoods ---

// CreateNeighborhood creates a neighborhood and adds the creator as a member.
func (s *Store) CreateNeighborhood(name, createdBy string) (*Neighborhood, error) {
	id := newID()
	code := newJoinCode()
	now := time.Now().UTC().Format(time.DateTime)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO neighborhoods (id, name, join_code, created_by, created_at) VALUES (?, ?, ?, ?, ?)",
		id, name, code, createdBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("store: create neighborhood: %w", err)
	}

	_, err = tx.Exec(
		"INSERT INTO neighborhood_members (neighborhood_id, user_id, joined_at) VALUES (?, ?, ?)",
		id, createdBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("store: add creator as member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit: %w", err)
	}

	return &Neighborhood{
		ID:        id,
		Name:      name,
		JoinCode:  code,
		CreatedBy: createdBy,
		CreatedAt: now,
	}, nil
}

// JoinNeighborhood adds a user to a neighborhood by join code.
// Returns the neighborhood or an error if the code is invalid.
func (s *Store) JoinNeighborhood(joinCode, userID string) (*Neighborhood, error) {
	n := &Neighborhood{}
	err := s.db.QueryRow(
		"SELECT id, name, join_code, created_by, created_at FROM neighborhoods WHERE join_code = ?",
		joinCode,
	).Scan(&n.ID, &n.Name, &n.JoinCode, &n.CreatedBy, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("store: invalid join code")
	}
	if err != nil {
		return nil, fmt.Errorf("store: lookup join code: %w", err)
	}

	_, err = s.db.Exec(
		"INSERT OR IGNORE INTO neighborhood_members (neighborhood_id, user_id) VALUES (?, ?)",
		n.ID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: join neighborhood: %w", err)
	}

	return n, nil
}

// GetUserNeighborhoods returns all neighborhoods a user belongs to.
func (s *Store) GetUserNeighborhoods(userID string) ([]Neighborhood, error) {
	rows, err := s.db.Query(`
		SELECT n.id, n.name, n.join_code, n.created_by, n.created_at
		FROM neighborhoods n
		JOIN neighborhood_members nm ON n.id = nm.neighborhood_id
		WHERE nm.user_id = ?
		ORDER BY n.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("store: get user neighborhoods: %w", err)
	}
	defer rows.Close()

	var result []Neighborhood
	for rows.Next() {
		var n Neighborhood
		if err := rows.Scan(&n.ID, &n.Name, &n.JoinCode, &n.CreatedBy, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan neighborhood: %w", err)
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

// GetNeighborhoodMembers returns all user IDs and emails in a neighborhood.
func (s *Store) GetNeighborhoodMembers(neighborhoodID string) ([]User, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.email, u.created_at
		FROM users u
		JOIN neighborhood_members nm ON u.id = nm.user_id
		WHERE nm.neighborhood_id = ?
		ORDER BY u.email
	`, neighborhoodID)
	if err != nil {
		return nil, fmt.Errorf("store: get neighborhood members: %w", err)
	}
	defer rows.Close()

	var result []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan member: %w", err)
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// IsNeighborhoodMember checks if a user is a member of a neighborhood.
func (s *Store) IsNeighborhoodMember(neighborhoodID, userID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM neighborhood_members WHERE neighborhood_id = ? AND user_id = ?",
		neighborhoodID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("store: check membership: %w", err)
	}
	return count > 0, nil
}

// GetNeighborhoodByID returns a neighborhood by ID. Returns nil if not found.
func (s *Store) GetNeighborhoodByID(id string) (*Neighborhood, error) {
	n := &Neighborhood{}
	err := s.db.QueryRow(
		"SELECT id, name, join_code, created_by, created_at FROM neighborhoods WHERE id = ?", id,
	).Scan(&n.ID, &n.Name, &n.JoinCode, &n.CreatedBy, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get neighborhood by id: %w", err)
	}
	return n, nil
}

// --- Listings ---

// CreateListing inserts a new listing and returns the generated ID.
func (s *Store) CreateListing(userID, neighborhoodID, title, description, category string, minTier int) (string, error) {
	id := newID()
	_, err := s.db.Exec(
		"INSERT INTO listings (id, user_id, neighborhood_id, title, description, category, min_tier) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, userID, neighborhoodID, title, description, category, minTier,
	)
	if err != nil {
		return "", fmt.Errorf("store: create listing: %w", err)
	}
	return id, nil
}

// GetListingByID returns a listing by ID. Returns nil if not found.
func (s *Store) GetListingByID(id string) (*Listing, error) {
	l := &Listing{}
	err := s.db.QueryRow(
		"SELECT id, user_id, neighborhood_id, title, description, category, min_tier, created_at FROM listings WHERE id = ?",
		id,
	).Scan(&l.ID, &l.UserID, &l.NeighborhoodID, &l.Title, &l.Description, &l.Category, &l.MinTier, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get listing by id: %w", err)
	}
	return l, nil
}

// UpdateListing updates an existing listing's fields.
func (s *Store) UpdateListing(id, title, description, category string, minTier int) error {
	_, err := s.db.Exec(
		"UPDATE listings SET title = ?, description = ?, category = ?, min_tier = ? WHERE id = ?",
		title, description, category, minTier, id,
	)
	if err != nil {
		return fmt.Errorf("store: update listing: %w", err)
	}
	return nil
}

// DeleteListing removes a listing by ID.
func (s *Store) DeleteListing(id string) error {
	_, err := s.db.Exec("DELETE FROM listings WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("store: delete listing: %w", err)
	}
	return nil
}

// GetUserListings returns all listings owned by a user in a neighborhood.
func (s *Store) GetUserListings(userID, neighborhoodID string) ([]Listing, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, neighborhood_id, title, description, category, min_tier, created_at
		FROM listings
		WHERE user_id = ? AND neighborhood_id = ?
		ORDER BY created_at DESC
	`, userID, neighborhoodID)
	if err != nil {
		return nil, fmt.Errorf("store: get user listings: %w", err)
	}
	defer rows.Close()

	var result []Listing
	for rows.Next() {
		var l Listing
		if err := rows.Scan(&l.ID, &l.UserID, &l.NeighborhoodID, &l.Title, &l.Description, &l.Category, &l.MinTier, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan listing: %w", err)
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

// GetVisibleListings returns listings in a neighborhood that the viewer can see,
// applying the same visibility rules as pkg/privacy.Visible via SQL.
//
// The WHERE clause mirrors the pure Go predicate:
//   - Owner always sees own listings (l.user_id = viewerID)
//   - Tier 0 listings visible to all neighborhood members (l.min_tier = 0)
//   - Otherwise, granted tier >= listing min tier
func (s *Store) GetVisibleListings(neighborhoodID, viewerID string) ([]Listing, error) {
	rows, err := s.db.Query(`
		SELECT l.id, l.user_id, l.neighborhood_id, l.title, l.description,
		       l.category, l.min_tier, l.created_at, u.email
		FROM listings l
		JOIN users u ON l.user_id = u.id
		LEFT JOIN trust_edges te ON te.owner_id = l.user_id AND te.neighbor_id = ?
		WHERE l.neighborhood_id = ?
		  AND (
		    l.user_id = ?
		    OR l.min_tier = 0
		    OR COALESCE(te.tier, -1) >= l.min_tier
		  )
		ORDER BY l.created_at DESC
	`, viewerID, neighborhoodID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("store: get visible listings: %w", err)
	}
	defer rows.Close()

	var result []Listing
	for rows.Next() {
		var l Listing
		if err := rows.Scan(&l.ID, &l.UserID, &l.NeighborhoodID, &l.Title, &l.Description,
			&l.Category, &l.MinTier, &l.CreatedAt, &l.OwnerEmail); err != nil {
			return nil, fmt.Errorf("store: scan visible listing: %w", err)
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

// --- Trust Edges ---

// SetTrust sets the trust tier that owner grants to neighbor.
// Uses upsert so it works for both initial setting and updates.
func (s *Store) SetTrust(ownerID, neighborID string, tier int) error {
	_, err := s.db.Exec(`
		INSERT INTO trust_edges (owner_id, neighbor_id, tier) VALUES (?, ?, ?)
		ON CONFLICT(owner_id, neighbor_id) DO UPDATE SET tier = excluded.tier
	`, ownerID, neighborID, tier)
	if err != nil {
		return fmt.Errorf("store: set trust: %w", err)
	}
	return nil
}

// GetTrust returns the trust tier that owner has granted to neighbor.
// Returns -1 (TierNone) if no trust edge exists.
func (s *Store) GetTrust(ownerID, neighborID string) (int, error) {
	var tier int
	err := s.db.QueryRow(
		"SELECT tier FROM trust_edges WHERE owner_id = ? AND neighbor_id = ?",
		ownerID, neighborID,
	).Scan(&tier)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("store: get trust: %w", err)
	}
	return tier, nil
}

// GetTrustEdgesForOwner returns all trust edges where the given user is the owner.
func (s *Store) GetTrustEdgesForOwner(ownerID string) ([]TrustEdge, error) {
	rows, err := s.db.Query(
		"SELECT owner_id, neighbor_id, tier FROM trust_edges WHERE owner_id = ? ORDER BY tier DESC",
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get trust edges: %w", err)
	}
	defer rows.Close()

	var result []TrustEdge
	for rows.Next() {
		var te TrustEdge
		if err := rows.Scan(&te.OwnerID, &te.NeighborID, &te.Tier); err != nil {
			return nil, fmt.Errorf("store: scan trust edge: %w", err)
		}
		result = append(result, te)
	}
	return result, rows.Err()
}

// RemoveTrust deletes a trust edge.
func (s *Store) RemoveTrust(ownerID, neighborID string) error {
	_, err := s.db.Exec(
		"DELETE FROM trust_edges WHERE owner_id = ? AND neighbor_id = ?",
		ownerID, neighborID,
	)
	if err != nil {
		return fmt.Errorf("store: remove trust: %w", err)
	}
	return nil
}

// --- Stats ---

// MemberCount returns the number of members in a neighborhood.
func (s *Store) MemberCount(neighborhoodID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM neighborhood_members WHERE neighborhood_id = ?",
		neighborhoodID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: member count: %w", err)
	}
	return count, nil
}

// ListingCount returns the number of listings in a neighborhood.
func (s *Store) ListingCount(neighborhoodID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM listings WHERE neighborhood_id = ?",
		neighborhoodID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("store: listing count: %w", err)
	}
	return count, nil
}

// Ping verifies the database connection is alive.
func (s *Store) Ping() error {
	return s.db.Ping()
}

// CleanExpiredSessions deletes sessions older than the given number of hours.
func (s *Store) CleanExpiredSessions(maxAgeHours int) (int64, error) {
	result, err := s.db.Exec(
		"DELETE FROM sessions WHERE created_at < datetime('now', ? || ' hours')",
		fmt.Sprintf("-%d", maxAgeHours),
	)
	if err != nil {
		return 0, fmt.Errorf("store: clean sessions: %w", err)
	}
	return result.RowsAffected()
}

// GetNeighborhoodByName looks up a neighborhood by name. Returns nil if not found.
func (s *Store) GetNeighborhoodByName(name string) (*Neighborhood, error) {
	n := &Neighborhood{}
	err := s.db.QueryRow(
		"SELECT id, name, join_code, created_by, created_at FROM neighborhoods WHERE name = ?", name,
	).Scan(&n.ID, &n.Name, &n.JoinCode, &n.CreatedBy, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get neighborhood by name: %w", err)
	}
	return n, nil
}
