package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	dbPath := filepath.Join(tmpDir, ".local", "share", "tusk", "tusk.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("Database file was not created at %s", dbPath)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	key := "test_key"
	value := "test_value"

	if err := store.Set(key, value); err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	retrieved, err := store.Get(key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %q, got %q", value, retrieved)
	}
}

func TestGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	value, err := store.Get("non_existent_key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if value != "" {
		t.Errorf("Expected empty string, got %q", value)
	}
}

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	key := "test_key"
	value := "test_value"

	if err := store.Set(key, value); err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	if err := store.Delete(key); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	retrieved, err := store.Get(key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if retrieved != "" {
		t.Errorf("Expected empty string after delete, got %q", retrieved)
	}
}

func TestAddPostToHistory(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	statusID := "123456789"

	if err := store.AddPostToHistory(statusID); err != nil {
		t.Fatalf("Failed to add post to history: %v", err)
	}

	retrieved, err := store.GetLastPostID()
	if err != nil {
		t.Fatalf("Failed to get last post ID: %v", err)
	}

	if retrieved != statusID {
		t.Errorf("Expected %q, got %q", statusID, retrieved)
	}
}

func TestGetLastPostIDEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	retrieved, err := store.GetLastPostID()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrieved != "" {
		t.Errorf("Expected empty string, got %q", retrieved)
	}
}

func TestRemovePostFromHistory(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	statusID := "123456789"

	if err := store.AddPostToHistory(statusID); err != nil {
		t.Fatalf("Failed to add post to history: %v", err)
	}

	if err := store.RemovePostFromHistory(statusID); err != nil {
		t.Fatalf("Failed to remove post from history: %v", err)
	}

	retrieved, err := store.GetLastPostID()
	if err != nil {
		t.Fatalf("Failed to get last post ID: %v", err)
	}

	if retrieved != "" {
		t.Errorf("Expected empty string after remove, got %q", retrieved)
	}
}

func TestClearAll(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	if err := store.Set("key1", "value1"); err != nil {
		t.Fatalf("Failed to set key1: %v", err)
	}

	if err := store.Set("key2", "value2"); err != nil {
		t.Fatalf("Failed to set key2: %v", err)
	}

	if err := store.AddPostToHistory("123456"); err != nil {
		t.Fatalf("Failed to add post to history: %v", err)
	}

	if err := store.ClearAll(); err != nil {
		t.Fatalf("Failed to clear all: %v", err)
	}

	value1, _ := store.Get("key1")
	if value1 != "" {
		t.Errorf("Expected key1 to be cleared, got %q", value1)
	}

	value2, _ := store.Get("key2")
	if value2 != "" {
		t.Errorf("Expected key2 to be cleared, got %q", value2)
	}

	lastPost, _ := store.GetLastPostID()
	if lastPost != "" {
		t.Errorf("Expected last post ID to be cleared, got %q", lastPost)
	}
}

func TestReplaceValue(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	key := "test_key"
	value1 := "value1"
	value2 := "value2"

	if err := store.Set(key, value1); err != nil {
		t.Fatalf("Failed to set initial value: %v", err)
	}

	if err := store.Set(key, value2); err != nil {
		t.Fatalf("Failed to replace value: %v", err)
	}

	retrieved, err := store.Get(key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if retrieved != value2 {
		t.Errorf("Expected %q, got %q", value2, retrieved)
	}
}

func TestPostHistoryStack(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add posts in order
	posts := []string{"post1", "post2", "post3"}
	for _, post := range posts {
		if err := store.AddPostToHistory(post); err != nil {
			t.Fatalf("Failed to add post %s: %v", post, err)
		}
	}

	// Last post should be post3
	lastPost, err := store.GetLastPostID()
	if err != nil {
		t.Fatalf("Failed to get last post: %v", err)
	}
	if lastPost != "post3" {
		t.Errorf("Expected last post 'post3', got %q", lastPost)
	}

	// Remove post3
	if err := store.RemovePostFromHistory("post3"); err != nil {
		t.Fatalf("Failed to remove post3: %v", err)
	}

	// Now last post should be post2
	lastPost, err = store.GetLastPostID()
	if err != nil {
		t.Fatalf("Failed to get last post: %v", err)
	}
	if lastPost != "post2" {
		t.Errorf("Expected last post 'post2', got %q", lastPost)
	}

	// Remove post2
	if err := store.RemovePostFromHistory("post2"); err != nil {
		t.Fatalf("Failed to remove post2: %v", err)
	}

	// Now last post should be post1
	lastPost, err = store.GetLastPostID()
	if err != nil {
		t.Fatalf("Failed to get last post: %v", err)
	}
	if lastPost != "post1" {
		t.Errorf("Expected last post 'post1', got %q", lastPost)
	}
}

func TestClearPostHistory(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add multiple posts
	posts := []string{"post1", "post2", "post3"}
	for _, post := range posts {
		if err := store.AddPostToHistory(post); err != nil {
			t.Fatalf("Failed to add post %s: %v", post, err)
		}
	}

	// Clear all history
	if err := store.ClearPostHistory(); err != nil {
		t.Fatalf("Failed to clear post history: %v", err)
	}

	// Should have no last post
	lastPost, err := store.GetLastPostID()
	if err != nil {
		t.Fatalf("Failed to get last post: %v", err)
	}
	if lastPost != "" {
		t.Errorf("Expected empty last post after clear, got %q", lastPost)
	}
}
