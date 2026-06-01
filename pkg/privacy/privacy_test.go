package privacy

import (
	"fmt"
	"testing"
)

func TestVisible_OwnerAlwaysSees(t *testing.T) {
	// Owner should see their own listings regardless of tier settings.
	for minTier := TierPublic; minTier <= TierInnerCircle; minTier++ {
		for grantedTier := TierNone; grantedTier <= TierInnerCircle; grantedTier++ {
			got := Visible("alice", "alice", minTier, grantedTier)
			if !got {
				t.Errorf("owner should always see own listing: minTier=%d grantedTier=%d", minTier, grantedTier)
			}
		}
	}
}

func TestVisible_TierMatrix(t *testing.T) {
	// Full matrix test: every combination of listing min tier vs granted tier.
	// ownerID != viewerID in all cases here.
	tests := []struct {
		listingMinTier int
		grantedTier    int
		want           bool
	}{
		// Public listings (tier 0) visible to everyone, even TierNone.
		{TierPublic, TierNone, true},
		{TierPublic, TierPublic, true},
		{TierPublic, TierNeighbor, true},
		{TierPublic, TierTrusted, true},
		{TierPublic, TierInnerCircle, true},

		// Neighbor listings (tier 1).
		{TierNeighbor, TierNone, false},
		{TierNeighbor, TierPublic, false},
		{TierNeighbor, TierNeighbor, true},
		{TierNeighbor, TierTrusted, true},
		{TierNeighbor, TierInnerCircle, true},

		// Trusted listings (tier 2).
		{TierTrusted, TierNone, false},
		{TierTrusted, TierPublic, false},
		{TierTrusted, TierNeighbor, false},
		{TierTrusted, TierTrusted, true},
		{TierTrusted, TierInnerCircle, true},

		// Inner Circle listings (tier 3).
		{TierInnerCircle, TierNone, false},
		{TierInnerCircle, TierPublic, false},
		{TierInnerCircle, TierNeighbor, false},
		{TierInnerCircle, TierTrusted, false},
		{TierInnerCircle, TierInnerCircle, true},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("listing=%s_granted=%s", TierName(tt.listingMinTier), TierName(tt.grantedTier))
		t.Run(name, func(t *testing.T) {
			got := Visible("owner", "viewer", tt.listingMinTier, tt.grantedTier)
			if got != tt.want {
				t.Errorf("Visible(owner, viewer, %d, %d) = %v, want %v",
					tt.listingMinTier, tt.grantedTier, got, tt.want)
			}
		})
	}
}

func TestTierName(t *testing.T) {
	tests := []struct {
		tier int
		want string
	}{
		{TierNone, "None"},
		{TierPublic, "Public"},
		{TierNeighbor, "Neighbor"},
		{TierTrusted, "Trusted"},
		{TierInnerCircle, "Inner Circle"},
		{99, "Unknown"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("tier_%d", tt.tier), func(t *testing.T) {
			got := TierName(tt.tier)
			if got != tt.want {
				t.Errorf("TierName(%d) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestValidTier(t *testing.T) {
	tests := []struct {
		tier int
		want bool
	}{
		{-1, false},
		{0, true},
		{1, true},
		{2, true},
		{3, true},
		{4, false},
		{100, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("tier_%d", tt.tier), func(t *testing.T) {
			got := ValidTier(tt.tier)
			if got != tt.want {
				t.Errorf("ValidTier(%d) = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}

func TestCanModify(t *testing.T) {
	tests := []struct {
		name    string
		ownerID string
		actorID string
		want    bool
	}{
		{"owner can modify", "alice", "alice", true},
		{"other user cannot modify", "alice", "bob", false},
		{"empty IDs match", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanModify(tt.ownerID, tt.actorID)
			if got != tt.want {
				t.Errorf("CanModify(%q, %q) = %v, want %v", tt.ownerID, tt.actorID, got, tt.want)
			}
		})
	}
}

func TestEffectiveTier(t *testing.T) {
	tests := []struct {
		name        string
		ownerID     string
		viewerID    string
		grantedTier int
		want        int
	}{
		{"owner gets max tier", "alice", "alice", TierNone, TierInnerCircle},
		{"owner gets max even with low grant", "alice", "alice", TierPublic, TierInnerCircle},
		{"other user gets granted tier", "alice", "bob", TierTrusted, TierTrusted},
		{"no trust edge stays at none", "alice", "bob", TierNone, TierNone},
		{"neighbor tier returned", "alice", "bob", TierNeighbor, TierNeighbor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffectiveTier(tt.ownerID, tt.viewerID, tt.grantedTier)
			if got != tt.want {
				t.Errorf("EffectiveTier(%q, %q, %d) = %d, want %d",
					tt.ownerID, tt.viewerID, tt.grantedTier, got, tt.want)
			}
		})
	}
}

func TestTierNames(t *testing.T) {
	names := TierNames()
	if len(names) != 4 {
		t.Fatalf("expected 4 tier names, got %d", len(names))
	}
	expected := []string{"Public", "Neighbor", "Trusted", "Inner Circle"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("TierNames()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestVisible_EmptyIDs(t *testing.T) {
	// Edge case: empty string IDs should still follow the rules.
	// Two empty strings match, so owner == viewer.
	if !Visible("", "", TierInnerCircle, TierNone) {
		t.Error("empty owner == empty viewer should return true")
	}
}

func TestVisible_SameIDDifferentCase(t *testing.T) {
	// IDs are case sensitive; "Alice" != "alice".
	if Visible("Alice", "alice", TierInnerCircle, TierNone) {
		t.Error("different case IDs should not be treated as the same owner")
	}
}
