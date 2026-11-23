package store

import (
	"context"
	"database/sql"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/supply"
)

type SupplyEditorArtwork struct {
	Width, Height, SizeBytes int
}

func (s *Store) SupplyEditorArtworkInfo(ctx context.Context, hash string, publishedOnly bool) (SupplyEditorArtwork, error) {
	var art SupplyEditorArtwork
	err := s.DB.QueryRowContext(ctx, `SELECT width,height,size_bytes FROM supply_editor_artwork
		WHERE sha256=? AND (published=TRUE OR ?=FALSE)`, hash, publishedOnly).Scan(&art.Width, &art.Height, &art.SizeBytes)
	return art, err
}

func (s *Store) SaveSupplyEditorArtwork(ctx context.Context, hash string, picture []byte, width, height int) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO supply_editor_artwork(sha256,picture,width,height,size_bytes)
		VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE sha256=VALUES(sha256)`, hash, picture, width, height, len(picture))
	return err
}

func (s *Store) PublishedSupplyEditorPicture(ctx context.Context, hash string) ([]byte, error) {
	var picture []byte
	err := s.DB.QueryRowContext(ctx, "SELECT picture FROM supply_editor_artwork WHERE sha256=? AND published=TRUE", hash).Scan(&picture)
	return picture, err
}

func (s *Store) publishSupplyEditorArtwork(ctx context.Context, tx *sql.Tx, definition supply.Definition) error {
	for _, id := range definition.AllArtwork() {
		if hash := supply.EditorArtworkHash(id); hash != "" {
			if _, err := tx.ExecContext(ctx, "UPDATE supply_editor_artwork SET published=TRUE WHERE sha256=?", hash); err != nil {
				return err
			}
		}
	}
	return nil
}
