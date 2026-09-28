package usecase

import (
	"testing"

	"uuid"
)

func TestEffectiveRelationIDs(t *testing.T) {
	existingID, addedID, missingID := uuid.New(), uuid.New(), uuid.New()
	added := effectiveRelationIDs([]uuid.UUID{existingID}, []uuid.UUID{existingID, addedID, addedID}, true)
	removed := effectiveRelationIDs([]uuid.UUID{existingID}, []uuid.UUID{existingID, missingID, existingID}, false)

	if len(added) != 1 || added[0] != addedID || len(removed) != 1 || removed[0] != existingID {
		t.Fatalf("added=%v removed=%v", added, removed)
	}
}
