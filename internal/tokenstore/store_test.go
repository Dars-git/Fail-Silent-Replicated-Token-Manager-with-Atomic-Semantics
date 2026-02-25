package tokenstore

import "testing"

func TestUpsertIfNewerAppliesFirstAndNewer(t *testing.T) {
	t.Parallel()

	store := New()
	id := int32(11)

	if !store.UpsertIfNewer(id, Value{WTS: "001-s1-000001", PartialVal: 1}) {
		t.Fatalf("expected first upsert to apply")
	}
	if !store.UpsertIfNewer(id, Value{WTS: "002-s1-000001", PartialVal: 2}) {
		t.Fatalf("expected newer upsert to apply")
	}

	got, ok := store.Get(id)
	if !ok {
		t.Fatalf("expected value in store")
	}
	if got.WTS != "002-s1-000001" || got.PartialVal != 2 {
		t.Fatalf("unexpected final value: %+v", got)
	}
}

func TestUpsertIfNewerRejectsOlderAndEmpty(t *testing.T) {
	t.Parallel()

	store := New()
	id := int32(12)
	store.Set(id, Value{WTS: "010-s1-000001", PartialVal: 10})

	if store.UpsertIfNewer(id, Value{WTS: "009-s1-000001", PartialVal: 9}) {
		t.Fatalf("expected older upsert to be rejected")
	}
	if store.UpsertIfNewer(id, Value{WTS: "", PartialVal: 0}) {
		t.Fatalf("expected empty timestamp upsert to be rejected")
	}

	got, _ := store.Get(id)
	if got.WTS != "010-s1-000001" || got.PartialVal != 10 {
		t.Fatalf("value changed after stale writes: %+v", got)
	}
}
