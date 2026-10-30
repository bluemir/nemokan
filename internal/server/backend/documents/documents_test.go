package documents

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/server/backend/auth"
	"github.com/bluemir/monosketch/internal/shape"
	"github.com/bluemir/monosketch/internal/util"
)

const clientIP = "192.0.2.1"

func newTestManager(t *testing.T, conf Config) (*Manager, *time.Time) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	m, err := New(ctx, &conf, db)
	require.NoError(t, err)
	now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	return m, &now
}

func content(t *testing.T, name string) string {
	manager := shape.NewManager()
	manager.Execute(shape.AddShape{Shape: shape.NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")})
	file := shape.NewFile(name, geo.Point{}, manager.Root().Serializable(true).(*shape.SerializableGroup), manager.ShapeConnector().Serializable())
	data, err := json.Marshal(file)
	require.NoError(t, err)
	return string(data)
}

func TestCreate_AnonymousAndOwned(t *testing.T) {
	m, _ := newTestManager(t, Config{})
	ctx := context.Background()

	anonymous, err := m.Create(ctx, "", content(t, "첫 프로젝트"), clientIP)
	require.NoError(t, err)
	assert.Len(t, anonymous.Id, idLength)
	assert.Equal(t, "첫 프로젝트", anonymous.Name)
	assert.Equal(t, 1, anonymous.Revision)
	assert.Empty(t, anonymous.Visibility)

	owned, err := m.Create(ctx, "alice", content(t, "mine"), clientIP)
	require.NoError(t, err)
	assert.Equal(t, VisibilityPrivate, owned.Visibility)
	assert.NotEqual(t, anonymous.Id, owned.Id)
}

func TestCreate_InvalidContent(t *testing.T) {
	m, _ := newTestManager(t, Config{})

	_, err := m.Create(context.Background(), "", `{"version":1}`, clientIP)
	assert.True(t, errors.Is(err, ErrInvalidContent))
}

func TestCreate_TooLarge(t *testing.T) {
	m, _ := newTestManager(t, Config{MaxSize: 10})

	_, err := m.Create(context.Background(), "", content(t, "big"), clientIP)
	assert.True(t, errors.Is(err, ErrTooLarge))
}

func TestCreate_RateLimit(t *testing.T) {
	m, _ := newTestManager(t, Config{RateLimit: RateLimitConfig{Create: 2, Window: util.Duration(time.Hour)}})
	ctx := context.Background()

	for range 2 {
		_, err := m.Create(ctx, "", content(t, "x"), clientIP)
		require.NoError(t, err)
	}
	_, err := m.Create(ctx, "", content(t, "x"), clientIP)
	assert.True(t, errors.Is(err, ErrRateLimited))

	_, err = m.Create(ctx, "", content(t, "x"), "192.0.2.2")
	assert.NoError(t, err)
}

func TestUpdate_RevisionConflict(t *testing.T) {
	m, _ := newTestManager(t, Config{})
	ctx := context.Background()
	document, err := m.Create(ctx, "", content(t, "v1"), clientIP)
	require.NoError(t, err)

	updated, err := m.Update(ctx, document.Id, "", 1, content(t, "v2"))
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Revision)
	assert.Equal(t, "v2", updated.Name)

	// Someone saves on top of revision 1, which is not the latest any more.
	_, err = m.Update(ctx, document.Id, "bob", 1, content(t, "stale"))
	assert.True(t, errors.Is(err, ErrRevisionConflict))

	got, err := m.Get(ctx, document.Id, "")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Revision)
	assert.Equal(t, "v2", got.Name)
}

func TestVisibility(t *testing.T) {
	m, _ := newTestManager(t, Config{})
	ctx := context.Background()
	document, err := m.Create(ctx, "alice", content(t, "mine"), clientIP)
	require.NoError(t, err)

	// Private: others cannot tell it from a missing document.
	_, err = m.Get(ctx, document.Id, "bob")
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	_, err = m.Get(ctx, document.Id, "")
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	_, err = m.SetVisibility(ctx, document.Id, "bob", VisibilityEdit)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	_, err = m.SetVisibility(ctx, document.Id, "alice", VisibilityView)
	require.NoError(t, err)
	got, err := m.Get(ctx, document.Id, "bob")
	require.NoError(t, err)
	assert.False(t, CanEdit(got, "bob"))
	_, err = m.Update(ctx, document.Id, "bob", 1, content(t, "bob's"))
	assert.True(t, errors.Is(err, auth.ErrForbidden))
	_, err = m.SetVisibility(ctx, document.Id, "bob", VisibilityEdit)
	assert.True(t, errors.Is(err, auth.ErrForbidden))

	_, err = m.SetVisibility(ctx, document.Id, "alice", VisibilityEdit)
	require.NoError(t, err)
	_, err = m.Update(ctx, document.Id, "", 1, content(t, "guest's"))
	assert.NoError(t, err)

	_, err = m.SetVisibility(ctx, document.Id, "alice", "public")
	assert.True(t, errors.Is(err, ErrInvalidVisibility))
}

func TestDelete(t *testing.T) {
	m, _ := newTestManager(t, Config{})
	ctx := context.Background()
	anonymous, err := m.Create(ctx, "", content(t, "anonymous"), clientIP)
	require.NoError(t, err)
	owned, err := m.Create(ctx, "alice", content(t, "mine"), clientIP)
	require.NoError(t, err)

	assert.True(t, errors.Is(m.Delete(ctx, anonymous.Id, ""), auth.ErrForbidden))
	assert.True(t, errors.Is(m.Delete(ctx, anonymous.Id, "alice"), auth.ErrForbidden))
	assert.NoError(t, m.Delete(ctx, owned.Id, "alice"))
	_, err = m.Get(ctx, owned.Id, "alice")
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestListOwned(t *testing.T) {
	m, now := newTestManager(t, Config{})
	ctx := context.Background()
	first, err := m.Create(ctx, "alice", content(t, "first"), clientIP)
	require.NoError(t, err)
	*now = now.Add(time.Minute)
	_, err = m.Create(ctx, "alice", content(t, "second"), clientIP)
	require.NoError(t, err)
	_, err = m.Create(ctx, "bob", content(t, "bob's"), clientIP)
	require.NoError(t, err)
	*now = now.Add(time.Minute)
	_, err = m.Update(ctx, first.Id, "alice", 1, content(t, "first again"))
	require.NoError(t, err)

	list, err := m.ListOwned(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "first again", list[0].Name)
	assert.Equal(t, "second", list[1].Name)
}

func TestRemoveExpired(t *testing.T) {
	m, now := newTestManager(t, Config{Retention: util.Duration(90 * 24 * time.Hour)})
	ctx := context.Background()
	accessed, err := m.Create(ctx, "", content(t, "accessed"), clientIP)
	require.NoError(t, err)
	forgotten, err := m.Create(ctx, "", content(t, "forgotten"), clientIP)
	require.NoError(t, err)
	owned, err := m.Create(ctx, "alice", content(t, "mine"), clientIP)
	require.NoError(t, err)

	*now = now.Add(60 * 24 * time.Hour)
	_, err = m.Get(ctx, accessed.Id, "")
	require.NoError(t, err)
	*now = now.Add(31 * 24 * time.Hour)

	removed, err := m.RemoveExpired(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)

	_, err = m.Get(ctx, forgotten.Id, "")
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	_, err = m.Get(ctx, accessed.Id, "")
	assert.NoError(t, err)
	_, err = m.Get(ctx, owned.Id, "alice")
	assert.NoError(t, err)
}
