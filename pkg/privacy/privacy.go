// Package privacy implements the trust tier visibility predicate for blockroll.
//
// Trust tiers form a fixed integer ladder:
//
//	Tier 0 (Public)       - any authenticated user in the same neighborhood
//	Tier 1 (Neighbor)     - default when you add someone
//	Tier 2 (Trusted)      - explicitly promoted
//	Tier 3 (Inner Circle) - closest trust level
//
// Trust is directional: owner assigns a tier to a neighbor unilaterally.
// If no trust edge exists, the viewer has an effective tier of -1.
package privacy

// Tier constants. Higher values mean more trust.
const (
	TierNone        = -1 // no trust edge exists
	TierPublic      = 0  // visible to any authenticated neighborhood member
	TierNeighbor    = 1  // default trust level
	TierTrusted     = 2  // explicitly promoted
	TierInnerCircle = 3  // closest trust
)

// TierName returns a human readable label for a tier value.
func TierName(tier int) string {
	switch tier {
	case TierNone:
		return "None"
	case TierPublic:
		return "Public"
	case TierNeighbor:
		return "Neighbor"
	case TierTrusted:
		return "Trusted"
	case TierInnerCircle:
		return "Inner Circle"
	default:
		return "Unknown"
	}
}

// ValidTier returns true if tier is a valid listing visibility tier (0 through 3).
func ValidTier(tier int) bool {
	return tier >= TierPublic && tier <= TierInnerCircle
}

// Visible determines whether a viewer can see a listing.
//
// Rules:
//   - The owner always sees their own listings (ownerID == viewerID).
//   - Otherwise, the viewer's granted tier must be >= the listing's minimum tier.
//   - Tier 0 listings are visible to any authenticated user in the neighborhood
//     (grantedTier can be TierNone for this, but we still require neighborhood membership,
//     which is enforced at the query level, not here).
//   - If no trust edge exists (grantedTier == TierNone == -1), the viewer sees
//     only tier 0 (public) listings.
func Visible(ownerID, viewerID string, listingMinTier, grantedTier int) bool {
	// Owner always sees their own listings.
	if ownerID == viewerID {
		return true
	}

	// For public listings (tier 0), anyone in the neighborhood can see them.
	// Even viewers with no trust edge (tier -1) see tier 0 listings,
	// because tier 0 means "any authenticated neighborhood member."
	if listingMinTier == TierPublic {
		return true
	}

	// For listings above tier 0, the viewer needs a trust edge with sufficient tier.
	// No trust edge (TierNone = -1) means they see nothing above tier 0.
	return grantedTier >= listingMinTier
}

// CanModify checks whether a user is allowed to edit or delete a listing.
// Only the listing owner can modify their own listings.
func CanModify(listingOwnerID, actorID string) bool {
	return listingOwnerID == actorID
}

// EffectiveTier returns the effective visibility tier for a viewer.
// If the viewer is the owner, they get max tier (InnerCircle).
// Otherwise, the granted tier is returned as given.
func EffectiveTier(ownerID, viewerID string, grantedTier int) int {
	if ownerID == viewerID {
		return TierInnerCircle
	}
	return grantedTier
}

// TierNames returns a slice of all valid tier names in order from 0 to 3.
func TierNames() []string {
	return []string{"Public", "Neighbor", "Trusted", "Inner Circle"}
}
