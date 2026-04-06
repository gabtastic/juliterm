package tmux

import "testing"

func TestValidateName_Valid(t *testing.T) {
	valid := []string{"my-session", "dev", "a", "session 1", "my_session"}
	for _, name := range valid {
		if err := validateName(name); err != nil {
			t.Errorf("validateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateName_Empty(t *testing.T) {
	for _, name := range []string{"", "   ", "\t"} {
		if err := validateName(name); err == nil {
			t.Errorf("validateName(%q) = nil, want error", name)
		}
	}
}

func TestValidateName_TmuxSpecials(t *testing.T) {
	for _, name := range []string{"a.b", "a:b", "a/b"} {
		if err := validateName(name); err == nil {
			t.Errorf("validateName(%q) = nil, want error for tmux special", name)
		}
	}
}

func TestValidateName_TooLong(t *testing.T) {
	long := make([]byte, 257)
	for i := range long {
		long[i] = 'a'
	}
	if err := validateName(string(long)); err == nil {
		t.Errorf("validateName(len=%d) = nil, want error", len(long))
	}
}

func TestValidateName_MaxLength(t *testing.T) {
	maxLen := make([]byte, 256)
	for i := range maxLen {
		maxLen[i] = 'a'
	}
	if err := validateName(string(maxLen)); err != nil {
		t.Errorf("validateName(len=256) = %v, want nil", err)
	}
}

func TestRenameSession(t *testing.T) {
	tree := NewSessionTree()
	tree.UpdateSession(&Session{ID: "$1", Name: "old-name", Windows: []*Window{}})

	tree.RenameSession("$1", "new-name")

	s := tree.GetSession("$1")
	if s == nil {
		t.Fatal("session not found after rename")
	}
	if s.Name != "new-name" {
		t.Errorf("session name = %q, want %q", s.Name, "new-name")
	}
}

func TestRenameSession_NotFound(t *testing.T) {
	tree := NewSessionTree()
	// Should not panic for missing session.
	tree.RenameSession("$999", "anything")
}

func TestRenameSession_NotifiesChange(t *testing.T) {
	tree := NewSessionTree()
	tree.UpdateSession(&Session{ID: "$1", Name: "old", Windows: []*Window{}})

	called := false
	tree.SetOnChange(func() { called = true })

	tree.RenameSession("$1", "new")
	if !called {
		t.Error("onChange was not called after rename")
	}
}

func TestRenameSession_NoChangeIfSameName(t *testing.T) {
	tree := NewSessionTree()
	tree.UpdateSession(&Session{ID: "$1", Name: "same", Windows: []*Window{}})

	called := false
	tree.SetOnChange(func() { called = true })

	tree.RenameSession("$1", "same")
	if called {
		t.Error("onChange was called when name didn't change")
	}
}
