package sharedkernel

import "testing"

func TestEntity(t *testing.T) {
	entity := NewEntity()
	if entity.IsZero() {
		t.Fatal("new entity should have id")
	}
	if !entity.Equals(NewEntityWithID(entity.ID())) {
		t.Fatal("entities with same id should be equal")
	}
	if entity.Equals(NewEntityWithID("other")) {
		t.Fatal("entities with different ids should not be equal")
	}
	if !NewEntityWithID("  abc  ").Equals(NewEntityWithID("abc")) {
		t.Fatal("entity id should be trimmed")
	}
}
