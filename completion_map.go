//go:build noboltdb

package bukanir

import "github.com/anacrolix/torrent/storage"

// newPieceCompletion keeps piece completion in memory.
func newPieceCompletion(string) (storage.PieceCompletion, error) {
	return storage.NewMapPieceCompletion(), nil
}
