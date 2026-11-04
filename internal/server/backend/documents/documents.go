// Package documents keeps the documents saved on the server. See ADR-0002 and ADR-0011.
package documents

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/bluemir/nemokan/internal/server/backend/auth"
	"github.com/bluemir/nemokan/internal/shape"
	"github.com/bluemir/nemokan/internal/util"
)

var (
	ErrTooLarge          = errors.New("document is too large")
	ErrRateLimited       = errors.New("too many documents created, try again later")
	ErrRevisionConflict  = errors.New("document was changed by someone else")
	ErrInvalidContent    = errors.New("document content is not a document file")
	ErrInvalidVisibility = errors.New("unknown visibility")
)

// idLength letters from crypto/rand make about 91 bits. An anonymous document is open to
// anyone with its URL, so its id must not be guessable.
const idLength = 16

// cleanupInterval is how often expired anonymous documents are removed.
const cleanupInterval = time.Hour

type Config struct {
	// MaxSize caps the content of a document.
	MaxSize util.Size `yaml:"maxSize"`
	// Retention is how long an anonymous document survives without being accessed.
	Retention util.Duration
	RateLimit RateLimitConfig `yaml:"rateLimit"`
}

type RateLimitConfig struct {
	// Create is the number of documents one client IP may create per Window.
	Create int
	Window util.Duration
}

// withDefaults fills the fields not set in the config file.
func (conf Config) withDefaults() Config {
	if conf.MaxSize <= 0 {
		conf.MaxSize = 1 << 20
	}
	if conf.Retention <= 0 {
		conf.Retention = util.Duration(90 * 24 * time.Hour)
	}
	if conf.RateLimit.Create <= 0 {
		conf.RateLimit.Create = 30
	}
	if conf.RateLimit.Window <= 0 {
		conf.RateLimit.Window = util.Duration(time.Hour)
	}
	return conf
}

// Visibility is who can see or edit an owned document. An anonymous document has none, as
// anyone with the URL can edit it.
type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityView    Visibility = "view"
	VisibilityEdit    Visibility = "edit"
)

type Document struct {
	Id   string `json:"id" gorm:"primaryKey;size:16"`
	Name string `json:"name"`
	// Content is the document file JSON. See ADR-0004.
	Content  string `json:"content" gorm:"type:text"`
	Revision int    `json:"revision"`
	// Owner is the user name, or empty for an anonymous document.
	Owner          string     `json:"owner" gorm:"index;size:256"`
	Visibility     Visibility `json:"visibility,omitempty" gorm:"size:16"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LastAccessedAt time.Time  `json:"lastAccessedAt" gorm:"index"`
}

// Summary is a document without its content, for lists.
type Summary struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"`
	Revision   int        `json:"revision"`
	Visibility Visibility `json:"visibility"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Manager struct {
	db      *gorm.DB
	conf    Config
	limiter *rateLimiter
	now     func() time.Time
}

// New also starts removing expired anonymous documents until ctx is done.
func New(ctx context.Context, conf *Config, db *gorm.DB) (*Manager, error) {
	if err := db.WithContext(ctx).AutoMigrate(&Document{}); err != nil {
		return nil, errors.WithStack(err)
	}
	m := &Manager{db: db, conf: conf.withDefaults(), now: time.Now}
	limiter, err := newRateLimiter(m.conf.RateLimit.Create, m.conf.RateLimit.Window.Std(), rateLimitMaxKeys, func() time.Time { return m.now() })
	if err != nil {
		return nil, err
	}
	m.limiter = limiter

	go m.runCleanup(ctx)
	return m, nil
}

// MaxRequestBytes bounds a request body: the content limit plus room for JSON escaping.
func (m *Manager) MaxRequestBytes() int64 {
	return int64(m.conf.MaxSize)*2 + 4096
}

// Create makes a document of content. It is owned by owner, or anonymous if owner is empty.
// clientIP is used for the rate limit.
func (m *Manager) Create(ctx context.Context, owner string, content string, clientIP string) (*Document, error) {
	file, err := m.parseContent(content)
	if err != nil {
		return nil, err
	}
	allowed, err := m.limiter.Allow(ctx, clientIP)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrRateLimited
	}

	now := m.now()
	document := &Document{
		Id:             util.RandomString(idLength),
		Name:           file.Name,
		Content:        content,
		Revision:       1,
		Owner:          owner,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastAccessedAt: now,
	}
	if owner != "" {
		document.Visibility = VisibilityPrivate
	}
	if err := m.db.WithContext(ctx).Create(document).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return document, nil
}

// Get returns the document for viewer, an empty viewer being a guest, and records the access.
func (m *Manager) Get(ctx context.Context, id string, viewer string) (*Document, error) {
	document, err := m.find(ctx, id, viewer)
	if err != nil {
		return nil, err
	}
	document.LastAccessedAt = m.now()
	if err := m.db.WithContext(ctx).Model(document).UpdateColumn("last_accessed_at", document.LastAccessedAt).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return document, nil
}

// Update replaces the content if revision is the current one, and returns the document with
// the next revision.
func (m *Manager) Update(ctx context.Context, id string, viewer string, revision int, content string) (*Document, error) {
	file, err := m.parseContent(content)
	if err != nil {
		return nil, err
	}
	document, err := m.find(ctx, id, viewer)
	if err != nil {
		return nil, err
	}
	if !CanEdit(document, viewer) {
		return nil, errors.WithStack(auth.ErrForbidden)
	}

	now := m.now()
	result := m.db.WithContext(ctx).Model(&Document{}).
		Where("id = ? AND revision = ?", id, revision).
		UpdateColumns(map[string]any{
			"name":             file.Name,
			"content":          content,
			"revision":         revision + 1,
			"updated_at":       now,
			"last_accessed_at": now,
		})
	if result.Error != nil {
		return nil, errors.WithStack(result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, errors.Wrapf(ErrRevisionConflict, "server revision is %d", document.Revision)
	}

	document.Name = file.Name
	document.Content = content
	document.Revision = revision + 1
	document.UpdatedAt = now
	document.LastAccessedAt = now
	return document, nil
}

// SetVisibility is only for the owner.
func (m *Manager) SetVisibility(ctx context.Context, id string, viewer string, visibility Visibility) (*Document, error) {
	switch visibility {
	case VisibilityPrivate, VisibilityView, VisibilityEdit:
	default:
		return nil, errors.Wrapf(ErrInvalidVisibility, "%q", visibility)
	}
	document, err := m.findOwned(ctx, id, viewer)
	if err != nil {
		return nil, err
	}
	document.Visibility = visibility
	if err := m.db.WithContext(ctx).Model(document).UpdateColumn("visibility", visibility).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return document, nil
}

// Delete is only for the owner. An anonymous document is removed only when it expires.
func (m *Manager) Delete(ctx context.Context, id string, viewer string) error {
	document, err := m.findOwned(ctx, id, viewer)
	if err != nil {
		return err
	}
	return errors.WithStack(m.db.WithContext(ctx).Delete(document).Error)
}

// ListOwned returns the documents of owner, the most recently updated first.
func (m *Manager) ListOwned(ctx context.Context, owner string) ([]Summary, error) {
	summaries := []Summary{}
	err := m.db.WithContext(ctx).Model(&Document{}).
		Where("owner = ?", owner).
		Order("updated_at DESC").
		Find(&summaries).Error
	return summaries, errors.WithStack(err)
}

// CanEdit reports whether viewer, an empty viewer being a guest, can change the document.
func CanEdit(document *Document, viewer string) bool {
	return document.Owner == "" || document.Owner == viewer || document.Visibility == VisibilityEdit
}

func canView(document *Document, viewer string) bool {
	return CanEdit(document, viewer) || document.Visibility == VisibilityView
}

// find returns a not found error for a document viewer cannot see, so a private document is
// not told apart from a missing one.
func (m *Manager) find(ctx context.Context, id string, viewer string) (*Document, error) {
	document := &Document{}
	if err := m.db.WithContext(ctx).Take(document, "id = ?", id).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	if !canView(document, viewer) {
		return nil, errors.WithStack(gorm.ErrRecordNotFound)
	}
	return document, nil
}

func (m *Manager) findOwned(ctx context.Context, id string, viewer string) (*Document, error) {
	document, err := m.find(ctx, id, viewer)
	if err != nil {
		return nil, err
	}
	if document.Owner == "" || document.Owner != viewer {
		return nil, errors.WithStack(auth.ErrForbidden)
	}
	return document, nil
}

func (m *Manager) parseContent(content string) (shape.File, error) {
	if int64(len(content)) > int64(m.conf.MaxSize) {
		return shape.File{}, errors.Wrapf(ErrTooLarge, "limit is %s", m.conf.MaxSize)
	}
	file, err := shape.ParseFile([]byte(content))
	if err != nil {
		return shape.File{}, errors.Wrap(ErrInvalidContent, err.Error())
	}
	return file, nil
}

// RemoveExpired removes the anonymous documents not accessed for the retention.
func (m *Manager) RemoveExpired(ctx context.Context) (int64, error) {
	result := m.db.WithContext(ctx).
		Where("owner = ? AND last_accessed_at < ?", "", m.now().Add(-m.conf.Retention.Std())).
		Delete(&Document{})
	return result.RowsAffected, errors.WithStack(result.Error)
}

func (m *Manager) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed, err := m.RemoveExpired(ctx)
			if err != nil {
				logrus.Errorf("remove expired documents: %+v", err)
				continue
			}
			if removed > 0 {
				logrus.Infof("removed %d expired documents", removed)
			}
		}
	}
}
