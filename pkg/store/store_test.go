package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zandenkane/blockroll/pkg/privacy"
)

// testStore creates a temporary database for testing.
func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNew_CreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	s, err := New(path)
	if err != nil {
		t.Fatalf("New(%q) error: %v", path, err)
	}
	s.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("database file not created at %s", path)
	}
}

func TestUserCRUD(t *testing.T) {
	s := testStore(t)

	// Create.
	id, err := s.CreateUser("test@example.com", "hash123")
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if id == "" {
		t.Fatal("CreateUser returned empty ID")
	}

	// Get by email.
	u, err := s.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if u == nil {
		t.Fatal("GetUserByEmail returned nil")
	}
	if u.ID != id {
		t.Errorf("ID mismatch: got %q, want %q", u.ID, id)
	}
	if u.Email != "test@example.com" {
		t.Errorf("Email mismatch: got %q", u.Email)
	}

	// Get by ID.
	u2, err := s.GetUserByID(id)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if u2 == nil || u2.Email != "test@example.com" {
		t.Error("GetUserByID returned wrong user or nil")
	}

	// Not found.
	u3, err := s.GetUserByEmail("nope@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if u3 != nil {
		t.Error("expected nil for unknown email")
	}
}

func TestSessionCRUD(t *testing.T) {
	s := testStore(t)

	uid, _ := s.CreateUser("sess@example.com", "hash")

	token, err := s.CreateSession(uid)
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}

	got, err := s.GetSession(token)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if got != uid {
		t.Errorf("GetSession: got %q, want %q", got, uid)
	}

	if err := s.DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}

	got2, err := s.GetSession(token)
	if err != nil {
		t.Fatalf("GetSession after delete error: %v", err)
	}
	if got2 != "" {
		t.Errorf("session still exists after delete: %q", got2)
	}
}

func TestNeighborhoodCreateAndJoin(t *testing.T) {
	s := testStore(t)

	alice, _ := s.CreateUser("alice@example.com", "hash")
	bob, _ := s.CreateUser("bob@example.com", "hash")

	// Alice creates a neighborhood.
	n, err := s.CreateNeighborhood("Elm Street", alice)
	if err != nil {
		t.Fatalf("CreateNeighborhood error: %v", err)
	}
	if n.JoinCode == "" {
		t.Fatal("join code is empty")
	}
	if len(n.JoinCode) != 6 {
		t.Errorf("join code length: got %d, want 6", len(n.JoinCode))
	}

	// Alice should be a member.
	isMember, err := s.IsNeighborhoodMember(n.ID, alice)
	if err != nil {
		t.Fatalf("IsNeighborhoodMember error: %v", err)
	}
	if !isMember {
		t.Error("creator should be a member")
	}

	// Bob joins by code.
	joined, err := s.JoinNeighborhood(n.JoinCode, bob)
	if err != nil {
		t.Fatalf("JoinNeighborhood error: %v", err)
	}
	if joined.ID != n.ID {
		t.Errorf("joined wrong neighborhood: got %q, want %q", joined.ID, n.ID)
	}

	// Both should show up in members.
	members, err := s.GetNeighborhoodMembers(n.ID)
	if err != nil {
		t.Fatalf("GetNeighborhoodMembers error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}

	// Invalid join code.
	_, err = s.JoinNeighborhood("ZZZZZZ", bob)
	if err == nil {
		t.Error("expected error for invalid join code")
	}
}

func TestListingCRUD(t *testing.T) {
	s := testStore(t)

	uid, _ := s.CreateUser("lister@example.com", "hash")
	n, _ := s.CreateNeighborhood("Test Hood", uid)

	// Create listing.
	lid, err := s.CreateListing(uid, n.ID, "Chainsaw", "16 inch Stihl", "tool", 1)
	if err != nil {
		t.Fatalf("CreateListing error: %v", err)
	}

	// Get by ID.
	l, err := s.GetListingByID(lid)
	if err != nil {
		t.Fatalf("GetListingByID error: %v", err)
	}
	if l.Title != "Chainsaw" {
		t.Errorf("title mismatch: got %q", l.Title)
	}

	// Update.
	if err := s.UpdateListing(lid, "Big Chainsaw", "20 inch Stihl", "tool", 2); err != nil {
		t.Fatalf("UpdateListing error: %v", err)
	}
	l2, _ := s.GetListingByID(lid)
	if l2.Title != "Big Chainsaw" || l2.MinTier != 2 {
		t.Errorf("update failed: title=%q minTier=%d", l2.Title, l2.MinTier)
	}

	// Delete.
	if err := s.DeleteListing(lid); err != nil {
		t.Fatalf("DeleteListing error: %v", err)
	}
	l3, _ := s.GetListingByID(lid)
	if l3 != nil {
		t.Error("listing still exists after delete")
	}
}

func TestTrustEdges(t *testing.T) {
	s := testStore(t)

	alice, _ := s.CreateUser("alice@example.com", "hash")
	bob, _ := s.CreateUser("bob@example.com", "hash")

	// No edge: should return -1.
	tier, err := s.GetTrust(alice, bob)
	if err != nil {
		t.Fatalf("GetTrust error: %v", err)
	}
	if tier != -1 {
		t.Errorf("expected -1 for no edge, got %d", tier)
	}

	// Set trust.
	if err := s.SetTrust(alice, bob, 2); err != nil {
		t.Fatalf("SetTrust error: %v", err)
	}
	tier2, _ := s.GetTrust(alice, bob)
	if tier2 != 2 {
		t.Errorf("expected tier 2, got %d", tier2)
	}

	// Update trust (upsert).
	if err := s.SetTrust(alice, bob, 3); err != nil {
		t.Fatalf("SetTrust update error: %v", err)
	}
	tier3, _ := s.GetTrust(alice, bob)
	if tier3 != 3 {
		t.Errorf("expected tier 3, got %d", tier3)
	}

	// Trust is directional.
	tierReverse, _ := s.GetTrust(bob, alice)
	if tierReverse != -1 {
		t.Errorf("reverse trust should be -1, got %d", tierReverse)
	}

	// Remove trust.
	if err := s.RemoveTrust(alice, bob); err != nil {
		t.Fatalf("RemoveTrust error: %v", err)
	}
	tierAfter, _ := s.GetTrust(alice, bob)
	if tierAfter != -1 {
		t.Errorf("expected -1 after remove, got %d", tierAfter)
	}
}

// TestVisibleListings_MatchesPredicate verifies that the SQL WHERE clause
// in GetVisibleListings produces the same results as the pure Go predicate
// in pkg/privacy.Visible.
func TestVisibleListings_MatchesPredicate(t *testing.T) {
	s := testStore(t)

	alice, _ := s.CreateUser("alice@example.com", "hash")
	bob, _ := s.CreateUser("bob@example.com", "hash")
	carol, _ := s.CreateUser("carol@example.com", "hash")

	n, _ := s.CreateNeighborhood("Test", alice)
	s.JoinNeighborhood(n.JoinCode, bob)
	s.JoinNeighborhood(n.JoinCode, carol)

	// Alice creates listings at each tier.
	var listingIDs [4]string
	for tier := 0; tier <= 3; tier++ {
		id, err := s.CreateListing(alice, n.ID, tierLabel(tier), "desc", "skill", tier)
		if err != nil {
			t.Fatalf("create listing tier %d: %v", tier, err)
		}
		listingIDs[tier] = id
	}

	// Bob has tier 1 trust from Alice.
	s.SetTrust(alice, bob, 1)
	// Carol has no trust edge from Alice.

	// Verify SQL results match the pure Go predicate for Bob.
	bobListings, err := s.GetVisibleListings(n.ID, bob)
	if err != nil {
		t.Fatalf("GetVisibleListings(bob) error: %v", err)
	}

	bobVisible := make(map[string]bool)
	for _, l := range bobListings {
		bobVisible[l.ID] = true
	}

	for tier := 0; tier <= 3; tier++ {
		goResult := privacy.Visible(alice, bob, tier, 1) // bob has granted tier 1
		sqlResult := bobVisible[listingIDs[tier]]
		if goResult != sqlResult {
			t.Errorf("tier %d: Go predicate says %v, SQL says %v (bob, grantedTier=1)",
				tier, goResult, sqlResult)
		}
	}

	// Verify SQL results match for Carol (no trust edge, effective tier -1).
	carolListings, err := s.GetVisibleListings(n.ID, carol)
	if err != nil {
		t.Fatalf("GetVisibleListings(carol) error: %v", err)
	}

	carolVisible := make(map[string]bool)
	for _, l := range carolListings {
		carolVisible[l.ID] = true
	}

	for tier := 0; tier <= 3; tier++ {
		goResult := privacy.Visible(alice, carol, tier, -1) // carol has no trust edge
		sqlResult := carolVisible[listingIDs[tier]]
		if goResult != sqlResult {
			t.Errorf("tier %d: Go predicate says %v, SQL says %v (carol, grantedTier=-1)",
				tier, goResult, sqlResult)
		}
	}

	// Verify Alice sees all her own listings.
	aliceListings, err := s.GetVisibleListings(n.ID, alice)
	if err != nil {
		t.Fatalf("GetVisibleListings(alice) error: %v", err)
	}
	if len(aliceListings) != 4 {
		t.Errorf("alice should see all 4 listings, got %d", len(aliceListings))
	}
}

func tierLabel(tier int) string {
	switch tier {
	case 0:
		return "Public Item"
	case 1:
		return "Neighbor Item"
	case 2:
		return "Trusted Item"
	case 3:
		return "Inner Circle Item"
	default:
		return "Unknown"
	}
}

func TestMemberCount(t *testing.T) {
	s := testStore(t)

	alice, _ := s.CreateUser("alice@example.com", "hash")
	bob, _ := s.CreateUser("bob@example.com", "hash")

	n, _ := s.CreateNeighborhood("Test", alice)

	count, err := s.MemberCount(n.ID)
	if err != nil {
		t.Fatalf("MemberCount error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 member after create, got %d", count)
	}

	s.JoinNeighborhood(n.JoinCode, bob)

	count2, err := s.MemberCount(n.ID)
	if err != nil {
		t.Fatalf("MemberCount error: %v", err)
	}
	if count2 != 2 {
		t.Errorf("expected 2 members after join, got %d", count2)
	}
}

func TestListingCount(t *testing.T) {
	s := testStore(t)

	uid, _ := s.CreateUser("counter@example.com", "hash")
	n, _ := s.CreateNeighborhood("Test", uid)

	count, err := s.ListingCount(n.ID)
	if err != nil {
		t.Fatalf("ListingCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 listings, got %d", count)
	}

	s.CreateListing(uid, n.ID, "Item1", "", "tool", 0)
	s.CreateListing(uid, n.ID, "Item2", "", "skill", 1)

	count2, err := s.ListingCount(n.ID)
	if err != nil {
		t.Fatalf("ListingCount error: %v", err)
	}
	if count2 != 2 {
		t.Errorf("expected 2 listings, got %d", count2)
	}
}

func TestPing(t *testing.T) {
	s := testStore(t)
	if err := s.Ping(); err != nil {
		t.Fatalf("Ping error: %v", err)
	}
}

func TestGetNeighborhoodByName(t *testing.T) {
	s := testStore(t)

	uid, _ := s.CreateUser("finder@example.com", "hash")
	created, _ := s.CreateNeighborhood("Oak Avenue", uid)

	found, err := s.GetNeighborhoodByName("Oak Avenue")
	if err != nil {
		t.Fatalf("GetNeighborhoodByName error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find neighborhood, got nil")
	}
	if found.ID != created.ID {
		t.Errorf("ID mismatch: got %q, want %q", found.ID, created.ID)
	}

	// Not found case.
	missing, err := s.GetNeighborhoodByName("Nonexistent")
	if err != nil {
		t.Fatalf("GetNeighborhoodByName error: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for unknown name")
	}
}

func TestGetUserListings(t *testing.T) {
	s := testStore(t)

	alice, _ := s.CreateUser("alice@example.com", "hash")
	bob, _ := s.CreateUser("bob@example.com", "hash")
	n, _ := s.CreateNeighborhood("Test", alice)
	s.JoinNeighborhood(n.JoinCode, bob)

	s.CreateListing(alice, n.ID, "Alice Tool", "", "tool", 0)
	s.CreateListing(bob, n.ID, "Bob Skill", "", "skill", 0)

	aliceListings, err := s.GetUserListings(alice, n.ID)
	if err != nil {
		t.Fatalf("GetUserListings error: %v", err)
	}
	if len(aliceListings) != 1 {
		t.Errorf("expected 1 listing for alice, got %d", len(aliceListings))
	}
	if len(aliceListings) > 0 && aliceListings[0].Title != "Alice Tool" {
		t.Errorf("wrong listing title: got %q", aliceListings[0].Title)
	}
}

func TestDuplicateJoinIgnored(t *testing.T) {
	s := testStore(t)

	alice, _ := s.CreateUser("alice@example.com", "hash")
	n, _ := s.CreateNeighborhood("Test", alice)

	// Alice is already a member from creation. Joining again should not error.
	_, err := s.JoinNeighborhood(n.JoinCode, alice)
	if err != nil {
		t.Fatalf("duplicate join should not error: %v", err)
	}

	count, _ := s.MemberCount(n.ID)
	if count != 1 {
		t.Errorf("expected 1 member after duplicate join, got %d", count)
	}
}
