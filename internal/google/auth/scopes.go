package auth

import "sort"

// Google OAuth 2.0 scope constants for Gkestral v0.1.
const (
	// ScopeGemini grants access to the Generative Language API.
	// Not included in DefaultV01Scopes because Gemini API uses API key auth.
	// Kept for future semantic retrieval / model tuning.
	ScopeGemini = "https://www.googleapis.com/auth/generative-language"

	// ScopeDriveReadOnly grants read-only access to Google Drive files.
	ScopeDriveReadOnly = "https://www.googleapis.com/auth/drive.readonly"

	// ScopeDriveFile grants per-file access to files created or opened by the app.
	ScopeDriveFile = "https://www.googleapis.com/auth/drive.file"

	// ScopeGmailReadOnly grants read-only access to Gmail messages and settings.
	ScopeGmailReadOnly = "https://www.googleapis.com/auth/gmail.readonly"

	// ScopeGmailCompose grants send-only access for creating and sending messages.
	ScopeGmailCompose = "https://www.googleapis.com/auth/gmail.compose"

	// ScopeUserInfoEmail grants access to the user's email address.
	ScopeUserInfoEmail = "https://www.googleapis.com/auth/userinfo.email"

	// ScopeUserInfoProfile grants access to the user's basic profile info.
	ScopeUserInfoProfile = "https://www.googleapis.com/auth/userinfo.profile"
)

// ScopeSet is an unordered set of OAuth scope strings.
// The zero value is an empty set ready to use.
type ScopeSet struct {
	m map[string]struct{}
}

// NewScopeSet creates a ScopeSet from the given scope strings.
func NewScopeSet(scopes ...string) ScopeSet {
	s := ScopeSet{m: make(map[string]struct{}, len(scopes))}
	for _, sc := range scopes {
		s.m[sc] = struct{}{}
	}
	return s
}

// Add inserts one or more scopes into the set.
func (s *ScopeSet) Add(scopes ...string) {
	if s.m == nil {
		s.m = make(map[string]struct{})
	}
	for _, sc := range scopes {
		s.m[sc] = struct{}{}
	}
}

// Remove deletes a scope from the set.
func (s *ScopeSet) Remove(scope string) {
	delete(s.m, scope)
}

// Contains reports whether the set contains the given scope.
func (s ScopeSet) Contains(scope string) bool {
	_, ok := s.m[scope]
	return ok
}

// Len returns the number of scopes in the set.
func (s ScopeSet) Len() int {
	return len(s.m)
}

// Slice returns a sorted slice of all scopes for deterministic serialization.
func (s ScopeSet) Slice() []string {
	out := make([]string, 0, len(s.m))
	for sc := range s.m {
		out = append(out, sc)
	}
	sort.Strings(out)
	return out
}

// Merge adds all scopes from other into this set.
func (s *ScopeSet) Merge(other ScopeSet) {
	if s.m == nil {
		s.m = make(map[string]struct{})
	}
	for sc := range other.m {
		s.m[sc] = struct{}{}
	}
}

// Equal reports whether two scope sets contain the same scopes.
func (s ScopeSet) Equal(other ScopeSet) bool {
	if len(s.m) != len(other.m) {
		return false
	}
	for sc := range s.m {
		if !other.Contains(sc) {
			return false
		}
	}
	return true
}

// ScopeProfile describes a named group of scopes for a specific capability.
type ScopeProfile struct {
	Name        string   // Human-readable name (e.g., "Drive Read")
	Description string   // What this profile grants
	Scopes      []string // The actual scope strings
	Required    bool     // Whether this profile is mandatory for the app
}

// DefaultV01Scopes returns the minimal scopes for Gkestral v0.1.
// Only userinfo.email is requested initially. Gemini API uses API key
// authentication, so the generative-language scope is not needed by default.
func DefaultV01Scopes() ScopeSet {
	return NewScopeSet(ScopeUserInfoEmail)
}

// V01WithDrive extends DefaultV01Scopes with Drive read-only access.
func V01WithDrive() ScopeSet {
	s := DefaultV01Scopes()
	s.Add(ScopeDriveReadOnly)
	return s
}

// V01WithGmail extends DefaultV01Scopes with Gmail read-only access.
func V01WithGmail() ScopeSet {
	s := DefaultV01Scopes()
	s.Add(ScopeGmailReadOnly)
	return s
}

// V01Full returns all v0.1 scopes: email + Drive read + Gmail read.
func V01Full() ScopeSet {
	s := DefaultV01Scopes()
	s.Add(ScopeDriveReadOnly, ScopeGmailReadOnly)
	return s
}

// NeedsReauth reports whether the requested scopes contain any scope
// not present in the current set. If true, the user must re-authenticate
// to grant additional permissions.
func NeedsReauth(current, requested ScopeSet) bool {
	for sc := range requested.m {
		if !current.Contains(sc) {
			return true
		}
	}
	return false
}

// MissingScopes returns the set of scopes in requested that are not in current.
func MissingScopes(current, requested ScopeSet) ScopeSet {
	missing := ScopeSet{m: make(map[string]struct{})}
	for sc := range requested.m {
		if !current.Contains(sc) {
			missing.m[sc] = struct{}{}
		}
	}
	return missing
}
