package session

import (
	"context"
	"fmt"
	"party-buddy/internal/db"
	"party-buddy/internal/session/storage"

	"github.com/jackc/pgx/v5"
)

type Manager struct {
	ctx     context.Context
	db      *db.DBPool
	storage storage.SyncStorage
}

func NewManager(ctx context.Context, db *db.DBPool) *Manager {
	return &Manager{
		ctx:     ctx,
		db:      db,
		storage: storage.SyncStorage{},
	}
}

// NewSession creates a new session.
//
// Assumes all values are valid.
func (m *Manager) NewSession(
	game *storage.Game,
	owner storage.ClientId,
	ownerNickname string,
	requireReady bool,
	playersMax int,
) (sid storage.SessionId, code storage.InviteCode, ownerId storage.PlayerId, err error) {
	m.storage.Atomically(func(s *storage.UnsafeStorage) {
		sid, code, ownerId, err = s.NewSession(game, owner, ownerNickname, requireReady, playersMax)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				s.RemoveSession(sid)
			}
		}()

		err = m.db.AcquireTx(m.ctx, func(tx pgx.Tx) error {
			if err = m.registerImage(tx, sid, game.ImageId); err != nil {
				return err
			}

			for _, task := range game.Tasks {
				if err = m.registerImage(tx, sid, task.ImageId()); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return
		}
	})

	return
}

func (m *Manager) registerImage(tx pgx.Tx, sid storage.SessionId, imageId storage.ImageId) error {
	if imageId.Valid {
		if err := db.CreateSessionImageRef(tx, m.ctx, sid.UUID(), imageId.UUID); err != nil {
			return fmt.Errorf("could not register an image (id %s) for session %s: %w", imageId, sid, err)
		}
	}

	return nil
}
