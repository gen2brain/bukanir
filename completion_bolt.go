//go:build !noboltdb

package bukanir

import "github.com/anacrolix/torrent/storage"

// newPieceCompletion keeps piece completion in a bbolt database in dir.
func newPieceCompletion(dir string) (storage.PieceCompletion, error) {
	return storage.NewBoltPieceCompletion(dir)
}
