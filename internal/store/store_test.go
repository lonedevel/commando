package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecentAndPresets(t *testing.T) {
	t.Setenv("COMMANDO_DATA_DIR", t.TempDir())
	s := Load()
	t0 := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	s.AddRecent("ls", "ls -l", t0)
	s.AddRecent("ls", "ls -a", t0.Add(time.Minute))
	s.AddRecent("ls", "ls -l", t0.Add(2*time.Minute)) // moves to the front
	s.AddRecent("grep", "grep -rn TODO .", t0.Add(3*time.Minute))
	for i := 0; i < MaxRecent+5; i++ {
		s.AddRecent("tar", "tar -cf a.tar "+string(rune('a'+i)), t0.Add(time.Duration(i)*time.Second))
	}
	s.SavePreset("ls", "long", "ls -l", t0)
	s.SavePreset("ls", "All", "ls -a", t0)
	s.SavePreset("ls", "LONG", "ls -lh", t0) // replaces "long"

	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	s = Load()

	presets, recent := s.For("ls")
	if len(presets) != 2 || presets[0].Name != "All" || presets[1].Line != "ls -lh" {
		t.Errorf("presets = %+v", presets)
	}
	if len(recent) != 2 || recent[0].Line != "ls -l" || recent[1].Line != "ls -a" {
		t.Errorf("recent = %+v", recent)
	}
	if _, r := s.For("tar"); len(r) != MaxRecent {
		t.Errorf("tar recent = %d, want %d", len(r), MaxRecent)
	}
	if all := s.AllRecent(2); len(all) != 2 || all[0].Line != "grep -rn TODO ." {
		t.Errorf("AllRecent = %+v", all)
	}

	s.DeletePreset("ls", "all")
	s.DeleteRecent("ls", "ls -a")
	presets, recent = s.For("ls")
	if len(presets) != 1 || len(recent) != 1 {
		t.Errorf("after delete: %+v %+v", presets, recent)
	}
	s.DeletePreset("ls", "long")
	s.DeleteRecent("ls", "ls -l")
	if _, ok := s.Commands["ls"]; ok {
		t.Error("empty command not pruned")
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COMMANDO_DATA_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "store.json"), []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Load()
	s.AddRecent("ls", "ls", time.Now())
	if err := s.Save(); err != nil {
		t.Fatalf("save after corrupt load: %v", err)
	}
}

func TestUnder(t *testing.T) {
	t.Setenv("COMMANDO_DATA_DIR", t.TempDir())
	s := Load()
	now := time.Now()
	s.AddRecent("git", "git --version", now.Add(-3*time.Minute))
	s.AddRecent("git commit", "git commit -a", now.Add(-time.Minute))
	s.AddRecent("git push", "git push", now.Add(-2*time.Minute))
	s.AddRecent("gitk", "gitk --all", now)
	s.SavePreset("git log", "pretty", "git log --oneline", now)
	presets, recent := s.Under("git", 2)
	if len(presets) != 1 || presets[0].Name != "pretty" {
		t.Errorf("presets = %+v", presets)
	}
	if len(recent) != 2 || recent[0].Line != "git commit -a" || recent[1].Line != "git push" {
		t.Errorf("recent = %+v", recent)
	}
}

func TestMax(t *testing.T) {
	t.Setenv("COMMANDO_DATA_DIR", t.TempDir())
	s := Load()
	s.Max = 3
	for i := 0; i < 6; i++ {
		s.AddRecent("ls", "ls -"+string(rune('a'+i)), time.Now().Add(time.Duration(i)*time.Minute))
	}
	if _, r := s.For("ls"); len(r) != 3 || r[0].Line != "ls -f" {
		t.Errorf("recent = %+v", r)
	}
}
