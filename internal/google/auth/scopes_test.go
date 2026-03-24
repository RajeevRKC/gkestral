package auth

import (
	"testing"
)

func TestNewScopeSet(t *testing.T) {
	s := NewScopeSet(ScopeUserInfoEmail, ScopeDriveReadOnly)
	if s.Len() != 2 {
		t.Errorf("Len() = %d, want 2", s.Len())
	}
	if !s.Contains(ScopeUserInfoEmail) {
		t.Error("should contain userinfo.email")
	}
	if !s.Contains(ScopeDriveReadOnly) {
		t.Error("should contain drive.readonly")
	}
}

func TestNewScopeSet_Empty(t *testing.T) {
	s := NewScopeSet()
	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
}

func TestNewScopeSet_Deduplicates(t *testing.T) {
	s := NewScopeSet(ScopeUserInfoEmail, ScopeUserInfoEmail, ScopeUserInfoEmail)
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1 (deduplication)", s.Len())
	}
}

func TestScopeSet_Add(t *testing.T) {
	s := NewScopeSet(ScopeUserInfoEmail)
	s.Add(ScopeDriveReadOnly, ScopeGmailReadOnly)
	if s.Len() != 3 {
		t.Errorf("Len() = %d, want 3", s.Len())
	}
}

func TestScopeSet_AddToZeroValue(t *testing.T) {
	var s ScopeSet
	s.Add(ScopeUserInfoEmail)
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1 after Add to zero-value", s.Len())
	}
}

func TestScopeSet_Remove(t *testing.T) {
	s := NewScopeSet(ScopeUserInfoEmail, ScopeDriveReadOnly)
	s.Remove(ScopeDriveReadOnly)
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1", s.Len())
	}
	if s.Contains(ScopeDriveReadOnly) {
		t.Error("should not contain drive.readonly after Remove")
	}
}

func TestScopeSet_RemoveNonexistent(t *testing.T) {
	s := NewScopeSet(ScopeUserInfoEmail)
	s.Remove("nonexistent")
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1", s.Len())
	}
}

func TestScopeSet_Contains(t *testing.T) {
	s := NewScopeSet(ScopeUserInfoEmail)
	if !s.Contains(ScopeUserInfoEmail) {
		t.Error("Contains should return true for present scope")
	}
	if s.Contains(ScopeDriveReadOnly) {
		t.Error("Contains should return false for absent scope")
	}
}

func TestScopeSet_ContainsZeroValue(t *testing.T) {
	var s ScopeSet
	if s.Contains(ScopeUserInfoEmail) {
		t.Error("Contains on zero-value should return false")
	}
}

func TestScopeSet_Slice(t *testing.T) {
	s := NewScopeSet(ScopeGmailReadOnly, ScopeUserInfoEmail, ScopeDriveReadOnly)
	sl := s.Slice()
	if len(sl) != 3 {
		t.Fatalf("Slice() len = %d, want 3", len(sl))
	}
	// Verify sorted order
	for i := 1; i < len(sl); i++ {
		if sl[i] < sl[i-1] {
			t.Errorf("Slice() not sorted: %s < %s", sl[i], sl[i-1])
		}
	}
}

func TestScopeSet_SliceEmpty(t *testing.T) {
	s := NewScopeSet()
	sl := s.Slice()
	if len(sl) != 0 {
		t.Errorf("Slice() len = %d, want 0", len(sl))
	}
}

func TestScopeSet_Merge(t *testing.T) {
	s1 := NewScopeSet(ScopeUserInfoEmail)
	s2 := NewScopeSet(ScopeDriveReadOnly, ScopeGmailReadOnly)
	s1.Merge(s2)
	if s1.Len() != 3 {
		t.Errorf("Len() = %d, want 3 after Merge", s1.Len())
	}
}

func TestScopeSet_MergeOverlapping(t *testing.T) {
	s1 := NewScopeSet(ScopeUserInfoEmail, ScopeDriveReadOnly)
	s2 := NewScopeSet(ScopeDriveReadOnly, ScopeGmailReadOnly)
	s1.Merge(s2)
	if s1.Len() != 3 {
		t.Errorf("Len() = %d, want 3 (overlapping merge)", s1.Len())
	}
}

func TestScopeSet_MergeIntoZeroValue(t *testing.T) {
	var s ScopeSet
	other := NewScopeSet(ScopeUserInfoEmail)
	s.Merge(other)
	if s.Len() != 1 {
		t.Errorf("Len() = %d, want 1 after Merge into zero value", s.Len())
	}
}

func TestScopeSet_Equal(t *testing.T) {
	s1 := NewScopeSet(ScopeUserInfoEmail, ScopeDriveReadOnly)
	s2 := NewScopeSet(ScopeDriveReadOnly, ScopeUserInfoEmail)
	if !s1.Equal(s2) {
		t.Error("Equal should be true for sets with same contents")
	}
}

func TestScopeSet_EqualDifferent(t *testing.T) {
	s1 := NewScopeSet(ScopeUserInfoEmail)
	s2 := NewScopeSet(ScopeDriveReadOnly)
	if s1.Equal(s2) {
		t.Error("Equal should be false for different sets")
	}
}

func TestScopeSet_EqualDifferentSize(t *testing.T) {
	s1 := NewScopeSet(ScopeUserInfoEmail)
	s2 := NewScopeSet(ScopeUserInfoEmail, ScopeDriveReadOnly)
	if s1.Equal(s2) {
		t.Error("Equal should be false for sets with different sizes")
	}
}

func TestDefaultV01Scopes(t *testing.T) {
	s := DefaultV01Scopes()
	if s.Len() != 1 {
		t.Errorf("DefaultV01Scopes Len() = %d, want 1", s.Len())
	}
	if !s.Contains(ScopeUserInfoEmail) {
		t.Error("DefaultV01Scopes should contain userinfo.email")
	}
	if s.Contains(ScopeGemini) {
		t.Error("DefaultV01Scopes should NOT contain generative-language (uses API key)")
	}
}

func TestV01WithDrive(t *testing.T) {
	s := V01WithDrive()
	if s.Len() != 2 {
		t.Errorf("V01WithDrive Len() = %d, want 2", s.Len())
	}
	if !s.Contains(ScopeUserInfoEmail) {
		t.Error("should contain userinfo.email")
	}
	if !s.Contains(ScopeDriveReadOnly) {
		t.Error("should contain drive.readonly")
	}
}

func TestV01WithGmail(t *testing.T) {
	s := V01WithGmail()
	if s.Len() != 2 {
		t.Errorf("V01WithGmail Len() = %d, want 2", s.Len())
	}
	if !s.Contains(ScopeGmailReadOnly) {
		t.Error("should contain gmail.readonly")
	}
}

func TestV01Full(t *testing.T) {
	s := V01Full()
	if s.Len() != 3 {
		t.Errorf("V01Full Len() = %d, want 3", s.Len())
	}
	if !s.Contains(ScopeUserInfoEmail) {
		t.Error("should contain userinfo.email")
	}
	if !s.Contains(ScopeDriveReadOnly) {
		t.Error("should contain drive.readonly")
	}
	if !s.Contains(ScopeGmailReadOnly) {
		t.Error("should contain gmail.readonly")
	}
}

func TestNeedsReauth(t *testing.T) {
	current := DefaultV01Scopes()
	requested := V01WithDrive()
	if !NeedsReauth(current, requested) {
		t.Error("NeedsReauth should be true when requesting Drive scope without it")
	}
}

func TestNeedsReauth_AlreadyHas(t *testing.T) {
	current := V01WithDrive()
	requested := V01WithDrive()
	if NeedsReauth(current, requested) {
		t.Error("NeedsReauth should be false when current covers requested")
	}
}

func TestNeedsReauth_Superset(t *testing.T) {
	current := V01Full()
	requested := V01WithDrive()
	if NeedsReauth(current, requested) {
		t.Error("NeedsReauth should be false when current is superset of requested")
	}
}

func TestMissingScopes(t *testing.T) {
	current := DefaultV01Scopes()
	requested := V01Full()
	missing := MissingScopes(current, requested)
	if missing.Len() != 2 {
		t.Errorf("MissingScopes Len() = %d, want 2", missing.Len())
	}
	if !missing.Contains(ScopeDriveReadOnly) {
		t.Error("missing should contain drive.readonly")
	}
	if !missing.Contains(ScopeGmailReadOnly) {
		t.Error("missing should contain gmail.readonly")
	}
}

func TestMissingScopes_NothingMissing(t *testing.T) {
	current := V01Full()
	requested := V01WithDrive()
	missing := MissingScopes(current, requested)
	if missing.Len() != 0 {
		t.Errorf("MissingScopes Len() = %d, want 0", missing.Len())
	}
}

func TestScopeProfile(t *testing.T) {
	profile := ScopeProfile{
		Name:        "Drive Read",
		Description: "Read-only access to Google Drive files",
		Scopes:      []string{ScopeDriveReadOnly},
		Required:    false,
	}
	if profile.Name != "Drive Read" {
		t.Errorf("Name = %q, want %q", profile.Name, "Drive Read")
	}
	if len(profile.Scopes) != 1 {
		t.Errorf("Scopes len = %d, want 1", len(profile.Scopes))
	}
}

func TestScopeConstants(t *testing.T) {
	// Verify scope strings are valid Google OAuth scope URLs
	scopes := []string{
		ScopeGemini,
		ScopeDriveReadOnly,
		ScopeDriveFile,
		ScopeGmailReadOnly,
		ScopeGmailCompose,
		ScopeUserInfoEmail,
		ScopeUserInfoProfile,
	}
	for _, sc := range scopes {
		if len(sc) < 30 {
			t.Errorf("scope %q seems too short for a Google OAuth scope URL", sc)
		}
	}
	if len(scopes) != 7 {
		t.Errorf("expected 7 scope constants, got %d", len(scopes))
	}
}
