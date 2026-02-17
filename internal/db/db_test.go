package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/shuhao/goviral/pkg/models"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New(:memory:) error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- my_posts tests ---

func TestUpsertPost_InsertAndUpdate(t *testing.T) {
	db := setupTestDB(t)

	post := &models.Post{
		Platform:       "x",
		PlatformPostID: "post-001",
		Content:        "Hello world",
		Likes:          10,
		Reposts:        5,
		Comments:       3,
		Impressions:    1000,
		PostedAt:       time.Now().Truncate(time.Second),
	}

	// Insert
	if err := db.UpsertPost(post); err != nil {
		t.Fatalf("UpsertPost() insert error = %v", err)
	}

	posts, err := db.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("GetAllPosts() returned %d posts, want 1", len(posts))
	}
	if posts[0].Content != "Hello world" {
		t.Errorf("Content = %q, want 'Hello world'", posts[0].Content)
	}
	if posts[0].Likes != 10 {
		t.Errorf("Likes = %d, want 10", posts[0].Likes)
	}
	if posts[0].Platform != "x" {
		t.Errorf("Platform = %q, want 'x'", posts[0].Platform)
	}

	// Update (same platform_post_id)
	post.Content = "Updated content"
	post.Likes = 50
	if err := db.UpsertPost(post); err != nil {
		t.Fatalf("UpsertPost() update error = %v", err)
	}

	posts, err = db.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts() after update error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("GetAllPosts() after upsert returned %d posts, want 1", len(posts))
	}
	if posts[0].Content != "Updated content" {
		t.Errorf("Content after update = %q, want 'Updated content'", posts[0].Content)
	}
	if posts[0].Likes != 50 {
		t.Errorf("Likes after update = %d, want 50", posts[0].Likes)
	}
}

func TestGetPostsByPlatform(t *testing.T) {
	db := setupTestDB(t)

	toInsert := []*models.Post{
		{Platform: "x", PlatformPostID: "x-001", Content: "X post 1", PostedAt: time.Now()},
		{Platform: "linkedin", PlatformPostID: "li-001", Content: "LinkedIn post", PostedAt: time.Now()},
		{Platform: "x", PlatformPostID: "x-002", Content: "X post 2", PostedAt: time.Now()},
	}
	for _, p := range toInsert {
		if err := db.UpsertPost(p); err != nil {
			t.Fatalf("UpsertPost() error = %v", err)
		}
	}

	xPosts, err := db.GetPostsByPlatform("x")
	if err != nil {
		t.Fatalf("GetPostsByPlatform('x') error = %v", err)
	}
	if len(xPosts) != 2 {
		t.Fatalf("GetPostsByPlatform('x') returned %d posts, want 2", len(xPosts))
	}
	for _, p := range xPosts {
		if p.Platform != "x" {
			t.Errorf("post.Platform = %q, want 'x'", p.Platform)
		}
	}

	liPosts, err := db.GetPostsByPlatform("linkedin")
	if err != nil {
		t.Fatalf("GetPostsByPlatform('linkedin') error = %v", err)
	}
	if len(liPosts) != 1 {
		t.Fatalf("GetPostsByPlatform('linkedin') returned %d posts, want 1", len(liPosts))
	}
	if liPosts[0].Content != "LinkedIn post" {
		t.Errorf("Content = %q, want 'LinkedIn post'", liPosts[0].Content)
	}
}

func TestGetAllPosts(t *testing.T) {
	db := setupTestDB(t)

	toInsert := []*models.Post{
		{Platform: "x", PlatformPostID: "x-001", Content: "First", PostedAt: time.Now()},
		{Platform: "linkedin", PlatformPostID: "li-001", Content: "Second", PostedAt: time.Now()},
	}
	for _, p := range toInsert {
		if err := db.UpsertPost(p); err != nil {
			t.Fatalf("UpsertPost() error = %v", err)
		}
	}

	all, err := db.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("GetAllPosts() returned %d posts, want 2", len(all))
	}
}

func TestGetAllPosts_Empty(t *testing.T) {
	db := setupTestDB(t)

	all, err := db.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts() error = %v", err)
	}
	if len(all) != 0 {
		t.Errorf("GetAllPosts() on empty db returned %d posts, want 0", len(all))
	}
}

// --- persona tests ---

func TestUpsertPersona_InsertAndUpdate(t *testing.T) {
	db := setupTestDB(t)

	persona := &models.Persona{
		Platform: "x",
		Profile: models.PersonaProfile{
			WritingTone:  "professional",
			VoiceSummary: "A tech leader.",
			CommonThemes: []string{"tech", "AI"},
		},
	}

	// Insert
	if err := db.UpsertPersona(persona); err != nil {
		t.Fatalf("UpsertPersona() insert error = %v", err)
	}

	got, err := db.GetPersona("x")
	if err != nil {
		t.Fatalf("GetPersona() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetPersona() returned nil, want persona")
	}
	if got.Profile.WritingTone != "professional" {
		t.Errorf("Profile.WritingTone = %q, want 'professional'", got.Profile.WritingTone)
	}
	if got.Profile.VoiceSummary != "A tech leader." {
		t.Errorf("Profile.VoiceSummary = %q, want 'A tech leader.'", got.Profile.VoiceSummary)
	}
	if len(got.Profile.CommonThemes) != 2 {
		t.Errorf("Profile.CommonThemes has %d items, want 2", len(got.Profile.CommonThemes))
	}
	if got.Platform != "x" {
		t.Errorf("Platform = %q, want 'x'", got.Platform)
	}

	// Update (same platform)
	persona.Profile.WritingTone = "casual"
	if err := db.UpsertPersona(persona); err != nil {
		t.Fatalf("UpsertPersona() update error = %v", err)
	}

	got, err = db.GetPersona("x")
	if err != nil {
		t.Fatalf("GetPersona() after update error = %v", err)
	}
	if got.Profile.WritingTone != "casual" {
		t.Errorf("Profile.WritingTone after update = %q, want 'casual'", got.Profile.WritingTone)
	}
}

func TestGetPersona_NotFound(t *testing.T) {
	db := setupTestDB(t)

	got, err := db.GetPersona("nonexistent")
	if err != nil {
		t.Fatalf("GetPersona() error = %v", err)
	}
	if got != nil {
		t.Errorf("GetPersona('nonexistent') = %v, want nil", got)
	}
}

func TestUpsertPersona_MultiplePlatforms(t *testing.T) {
	db := setupTestDB(t)

	xPersona := &models.Persona{
		Platform: "x",
		Profile:  models.PersonaProfile{WritingTone: "witty"},
	}
	liPersona := &models.Persona{
		Platform: "linkedin",
		Profile:  models.PersonaProfile{WritingTone: "professional"},
	}

	if err := db.UpsertPersona(xPersona); err != nil {
		t.Fatalf("UpsertPersona(x) error = %v", err)
	}
	if err := db.UpsertPersona(liPersona); err != nil {
		t.Fatalf("UpsertPersona(linkedin) error = %v", err)
	}

	gotX, err := db.GetPersona("x")
	if err != nil {
		t.Fatalf("GetPersona('x') error = %v", err)
	}
	if gotX.Profile.WritingTone != "witty" {
		t.Errorf("x persona WritingTone = %q, want 'witty'", gotX.Profile.WritingTone)
	}

	gotLI, err := db.GetPersona("linkedin")
	if err != nil {
		t.Fatalf("GetPersona('linkedin') error = %v", err)
	}
	if gotLI.Profile.WritingTone != "professional" {
		t.Errorf("linkedin persona WritingTone = %q, want 'professional'", gotLI.Profile.WritingTone)
	}
}

// --- trending_posts tests ---

func TestUpsertTrendingPost_InsertAndUpdate(t *testing.T) {
	db := setupTestDB(t)

	tp := &models.TrendingPost{
		Platform:       "x",
		PlatformPostID: "tp-001",
		AuthorUsername: "guru",
		AuthorName:     "Guru",
		Content:        "Trending content",
		Likes:          5000,
		Reposts:        1000,
		Comments:       200,
		Impressions:    100000,
		NicheTags:      []string{"tech", "AI"},
		PostedAt:       time.Now().Truncate(time.Second),
	}

	// Insert
	if err := db.UpsertTrendingPost(tp); err != nil {
		t.Fatalf("UpsertTrendingPost() insert error = %v", err)
	}

	posts, err := db.GetTrendingPosts("", 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("GetTrendingPosts() returned %d, want 1", len(posts))
	}
	if posts[0].Content != "Trending content" {
		t.Errorf("Content = %q, want 'Trending content'", posts[0].Content)
	}
	if len(posts[0].NicheTags) != 2 {
		t.Errorf("NicheTags = %v, want 2 tags", posts[0].NicheTags)
	}
	if posts[0].AuthorUsername != "guru" {
		t.Errorf("AuthorUsername = %q, want 'guru'", posts[0].AuthorUsername)
	}

	// Update
	tp.Likes = 10000
	tp.NicheTags = []string{"tech", "AI", "startups"}
	if err := db.UpsertTrendingPost(tp); err != nil {
		t.Fatalf("UpsertTrendingPost() update error = %v", err)
	}

	posts, err = db.GetTrendingPosts("", 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts() after update error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post after upsert, got %d", len(posts))
	}
	if posts[0].Likes != 10000 {
		t.Errorf("Likes after update = %d, want 10000", posts[0].Likes)
	}
	if len(posts[0].NicheTags) != 3 {
		t.Errorf("NicheTags after update = %v, want 3 tags", posts[0].NicheTags)
	}
}

func TestGetTrendingPosts_PlatformFilter(t *testing.T) {
	db := setupTestDB(t)

	tps := []*models.TrendingPost{
		{Platform: "x", PlatformPostID: "tp-x-001", Content: "X trending", Likes: 100, NicheTags: []string{"tech"}, PostedAt: time.Now()},
		{Platform: "linkedin", PlatformPostID: "tp-li-001", Content: "LI trending", Likes: 200, NicheTags: []string{"leadership"}, PostedAt: time.Now()},
		{Platform: "x", PlatformPostID: "tp-x-002", Content: "X trending 2", Likes: 300, NicheTags: []string{"AI"}, PostedAt: time.Now()},
	}
	for _, tp := range tps {
		if err := db.UpsertTrendingPost(tp); err != nil {
			t.Fatalf("UpsertTrendingPost() error = %v", err)
		}
	}

	xPosts, err := db.GetTrendingPosts("x", 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts('x') error = %v", err)
	}
	if len(xPosts) != 2 {
		t.Fatalf("GetTrendingPosts('x') returned %d, want 2", len(xPosts))
	}
	for _, p := range xPosts {
		if p.Platform != "x" {
			t.Errorf("post.Platform = %q, want 'x'", p.Platform)
		}
	}

	liPosts, err := db.GetTrendingPosts("linkedin", 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts('linkedin') error = %v", err)
	}
	if len(liPosts) != 1 {
		t.Fatalf("GetTrendingPosts('linkedin') returned %d, want 1", len(liPosts))
	}
}

func TestGetTrendingPosts_OrderedByLikes(t *testing.T) {
	db := setupTestDB(t)

	tps := []*models.TrendingPost{
		{Platform: "x", PlatformPostID: "tp-001", Content: "Low", Likes: 50, NicheTags: []string{}, PostedAt: time.Now()},
		{Platform: "x", PlatformPostID: "tp-002", Content: "High", Likes: 500, NicheTags: []string{}, PostedAt: time.Now()},
		{Platform: "x", PlatformPostID: "tp-003", Content: "Mid", Likes: 200, NicheTags: []string{}, PostedAt: time.Now()},
	}
	for _, tp := range tps {
		if err := db.UpsertTrendingPost(tp); err != nil {
			t.Fatalf("UpsertTrendingPost() error = %v", err)
		}
	}

	posts, err := db.GetTrendingPosts("", 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts() error = %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("GetTrendingPosts() returned %d, want 3", len(posts))
	}
	// Should be ordered by likes DESC
	if posts[0].Likes != 500 {
		t.Errorf("posts[0].Likes = %d, want 500 (highest)", posts[0].Likes)
	}
	if posts[1].Likes != 200 {
		t.Errorf("posts[1].Likes = %d, want 200", posts[1].Likes)
	}
	if posts[2].Likes != 50 {
		t.Errorf("posts[2].Likes = %d, want 50 (lowest)", posts[2].Likes)
	}
}

func TestGetTrendingPosts_WithLimit(t *testing.T) {
	db := setupTestDB(t)

	for i := 0; i < 5; i++ {
		tp := &models.TrendingPost{
			Platform:       "x",
			PlatformPostID: fmt.Sprintf("tp-%d", i),
			Content:        fmt.Sprintf("Post %d", i),
			Likes:          i * 100,
			NicheTags:      []string{"tech"},
			PostedAt:       time.Now(),
		}
		if err := db.UpsertTrendingPost(tp); err != nil {
			t.Fatalf("UpsertTrendingPost() error = %v", err)
		}
	}

	posts, err := db.GetTrendingPosts("", 3)
	if err != nil {
		t.Fatalf("GetTrendingPosts() error = %v", err)
	}
	if len(posts) != 3 {
		t.Errorf("GetTrendingPosts(limit=3) returned %d, want 3", len(posts))
	}
}

func TestGetTrendingPostByID(t *testing.T) {
	db := setupTestDB(t)

	tp := &models.TrendingPost{
		Platform:       "x",
		PlatformPostID: "tp-001",
		AuthorUsername: "guru",
		AuthorName:     "Guru",
		Content:        "Trending test",
		Likes:          100,
		NicheTags:      []string{"tech"},
		PostedAt:       time.Now(),
	}
	if err := db.UpsertTrendingPost(tp); err != nil {
		t.Fatalf("UpsertTrendingPost() error = %v", err)
	}

	// Fetch by ID (auto-increment starts at 1)
	got, err := db.GetTrendingPostByID(1)
	if err != nil {
		t.Fatalf("GetTrendingPostByID(1) error = %v", err)
	}
	if got == nil {
		t.Fatal("GetTrendingPostByID(1) returned nil")
	}
	if got.PlatformPostID != "tp-001" {
		t.Errorf("PlatformPostID = %q, want 'tp-001'", got.PlatformPostID)
	}
	if got.AuthorUsername != "guru" {
		t.Errorf("AuthorUsername = %q, want 'guru'", got.AuthorUsername)
	}
	if len(got.NicheTags) != 1 || got.NicheTags[0] != "tech" {
		t.Errorf("NicheTags = %v, want [tech]", got.NicheTags)
	}
}

func TestGetTrendingPostByID_NotFound(t *testing.T) {
	db := setupTestDB(t)

	got, err := db.GetTrendingPostByID(999)
	if err != nil {
		t.Fatalf("GetTrendingPostByID(999) error = %v", err)
	}
	if got != nil {
		t.Errorf("GetTrendingPostByID(999) = %v, want nil", got)
	}
}

// --- generated_content tests ---

func TestInsertGeneratedContent(t *testing.T) {
	db := setupTestDB(t)

	gc := &models.GeneratedContent{
		SourceTrendingID: 1,
		TargetPlatform:   "x",
		OriginalContent:  "Original trending post",
		GeneratedContent: "Rewritten viral content",
		PersonaID:        1,
		PromptUsed:       "test prompt",
		Status:           "draft",
	}

	id, err := db.InsertGeneratedContent(gc)
	if err != nil {
		t.Fatalf("InsertGeneratedContent() error = %v", err)
	}
	if id <= 0 {
		t.Errorf("InsertGeneratedContent() returned id=%d, want > 0", id)
	}
}

func TestGetGeneratedContent(t *testing.T) {
	db := setupTestDB(t)

	gcs := []*models.GeneratedContent{
		{SourceTrendingID: 1, TargetPlatform: "x", OriginalContent: "orig1", GeneratedContent: "gen1", Status: "draft"},
		{SourceTrendingID: 2, TargetPlatform: "x", OriginalContent: "orig2", GeneratedContent: "gen2", Status: "approved"},
		{SourceTrendingID: 3, TargetPlatform: "linkedin", OriginalContent: "orig3", GeneratedContent: "gen3", Status: "draft"},
	}
	for _, gc := range gcs {
		if _, err := db.InsertGeneratedContent(gc); err != nil {
			t.Fatalf("InsertGeneratedContent() error = %v", err)
		}
	}

	// All content
	all, err := db.GetGeneratedContent("", 0)
	if err != nil {
		t.Fatalf("GetGeneratedContent('', 0) error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("GetGeneratedContent('', 0) returned %d, want 3", len(all))
	}

	// Filter by status
	drafts, err := db.GetGeneratedContent("draft", 0)
	if err != nil {
		t.Fatalf("GetGeneratedContent('draft', 0) error = %v", err)
	}
	if len(drafts) != 2 {
		t.Errorf("GetGeneratedContent('draft', 0) returned %d, want 2", len(drafts))
	}

	approved, err := db.GetGeneratedContent("approved", 0)
	if err != nil {
		t.Fatalf("GetGeneratedContent('approved', 0) error = %v", err)
	}
	if len(approved) != 1 {
		t.Errorf("GetGeneratedContent('approved', 0) returned %d, want 1", len(approved))
	}

	// With limit
	limited, err := db.GetGeneratedContent("", 2)
	if err != nil {
		t.Fatalf("GetGeneratedContent('', 2) error = %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("GetGeneratedContent('', 2) returned %d, want 2", len(limited))
	}
}

func TestGetGeneratedContentByID(t *testing.T) {
	db := setupTestDB(t)

	gc := &models.GeneratedContent{
		SourceTrendingID: 1,
		TargetPlatform:   "x",
		OriginalContent:  "original",
		GeneratedContent: "generated",
		PersonaID:        1,
		PromptUsed:       "prompt",
		Status:           "draft",
	}
	id, err := db.InsertGeneratedContent(gc)
	if err != nil {
		t.Fatalf("InsertGeneratedContent() error = %v", err)
	}

	got, err := db.GetGeneratedContentByID(id)
	if err != nil {
		t.Fatalf("GetGeneratedContentByID(%d) error = %v", id, err)
	}
	if got == nil {
		t.Fatalf("GetGeneratedContentByID(%d) returned nil", id)
	}
	if got.GeneratedContent != "generated" {
		t.Errorf("GeneratedContent = %q, want 'generated'", got.GeneratedContent)
	}
	if got.OriginalContent != "original" {
		t.Errorf("OriginalContent = %q, want 'original'", got.OriginalContent)
	}
	if got.Status != "draft" {
		t.Errorf("Status = %q, want 'draft'", got.Status)
	}
}

func TestGetGeneratedContentByID_NotFound(t *testing.T) {
	db := setupTestDB(t)

	got, err := db.GetGeneratedContentByID(999)
	if err != nil {
		t.Fatalf("GetGeneratedContentByID(999) error = %v", err)
	}
	if got != nil {
		t.Errorf("GetGeneratedContentByID(999) = %v, want nil", got)
	}
}

func TestUpdateGeneratedContentStatus(t *testing.T) {
	db := setupTestDB(t)

	gc := &models.GeneratedContent{
		SourceTrendingID: 1,
		TargetPlatform:   "x",
		OriginalContent:  "original",
		GeneratedContent: "generated",
		Status:           "draft",
	}
	id, err := db.InsertGeneratedContent(gc)
	if err != nil {
		t.Fatalf("InsertGeneratedContent() error = %v", err)
	}

	// Update draft -> approved
	if err := db.UpdateGeneratedContentStatus(id, "approved"); err != nil {
		t.Fatalf("UpdateGeneratedContentStatus('approved') error = %v", err)
	}

	got, err := db.GetGeneratedContentByID(id)
	if err != nil {
		t.Fatalf("GetGeneratedContentByID() error = %v", err)
	}
	if got.Status != "approved" {
		t.Errorf("Status after first update = %q, want 'approved'", got.Status)
	}

	// Update approved -> posted
	if err := db.UpdateGeneratedContentStatus(id, "posted"); err != nil {
		t.Fatalf("UpdateGeneratedContentStatus('posted') error = %v", err)
	}

	got, err = db.GetGeneratedContentByID(id)
	if err != nil {
		t.Fatalf("GetGeneratedContentByID() error = %v", err)
	}
	if got.Status != "posted" {
		t.Errorf("Status after second update = %q, want 'posted'", got.Status)
	}
}

func TestDatabaseNew_CreatesSchema(t *testing.T) {
	db := setupTestDB(t)

	// Verify all tables exist by querying them
	tables := []string{"my_posts", "persona", "trending_posts", "generated_content"}
	for _, table := range tables {
		rows, err := db.conn.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 1", table))
		if err != nil {
			t.Errorf("table %q should exist but query failed: %v", table, err)
		} else {
			rows.Close()
		}
	}
}
