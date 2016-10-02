package torrent2http

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	lt "github.com/gen2brain/libtorrent-go"
)

type torrentFS struct {
	handle         lt.TorrentHandle
	info           lt.TorrentInfo
	priorities     map[int]int
	openedFiles    []*torrentFile
	lastOpenedFile *torrentFile
	shuttingDown   bool
	fileCounter    int
	progresses     lt.Std_vector_size_type
}

type torrentFile struct {
	tfs        *torrentFS
	num        int
	closed     bool
	savePath   string
	fileEntry  lt.FileEntry
	index      int
	filePtr    *os.File
	downloaded int64
	progress   float32
}

type torrentDir struct {
	tfs         *torrentFS
	entriesRead int
}

var (
	errFileNotFound = errors.New("File is not found")
	errInvalidIndex = errors.New("No file with such index")
)

func newTorrentFS(handle lt.TorrentHandle, startIndex int) *torrentFS {
	tfs := torrentFS{
		handle:     handle,
		priorities: make(map[int]int),
	}

	go func() {
		tfs.waitForMetadata()

		if startIndex < 0 {
			startIndex = tfs.FindLargestFileIndex()
			if config.Verbose {
				log.Printf("T2HTTP: Largest file index: %d", startIndex)
			}
		} else {
			if config.Verbose {
				log.Printf("T2HTTP: Start index: %d", startIndex)
			}
		}

		for i := 0; i < tfs.TorrentInfo().NumFiles(); i++ {
			if startIndex == i {
				tfs.setPriority(i, 1)
			} else {
				tfs.setPriority(i, 0)
			}
		}
	}()

	return &tfs
}

func (tfs *torrentFS) Shutdown() {
	tfs.shuttingDown = true

	if len(tfs.openedFiles) > 0 {
		if config.Verbose {
			log.Printf("T2HTTP: Closing %d opened file(s)", len(tfs.openedFiles))
		}
		for _, f := range tfs.openedFiles {
			f.Close()
		}
	}
}

func (tfs *torrentFS) LastOpenedFile() *torrentFile {
	return tfs.lastOpenedFile
}

func (tfs *torrentFS) addOpenedFile(file *torrentFile) {
	tfs.openedFiles = append(tfs.openedFiles, file)
}

func (tfs *torrentFS) setPriority(index int, priority int) {
	if val, ok := tfs.priorities[index]; !ok || val != priority {
		if config.Verbose {
			log.Printf("T2HTTP: Setting %s priority to %d", tfs.info.FileAt(index).GetPath(), priority)
		}

		tfs.priorities[index] = priority
		tfs.handle.FilePriority(index, priority)
	}
}

func (tfs *torrentFS) findOpenedFile(file *torrentFile) int {
	for i, f := range tfs.openedFiles {
		if f == file {
			return i
		}
	}

	return -1
}

func (tfs *torrentFS) removeOpenedFile(file *torrentFile) {
	pos := tfs.findOpenedFile(file)
	if pos >= 0 {
		tfs.openedFiles = append(tfs.openedFiles[:pos], tfs.openedFiles[pos+1:]...)
	}
}

func (tfs *torrentFS) FindLargestFileIndex() int {
	index := 0
	files := tfs.Files()

	for _, f := range files {
		if f.Size() > files[index].Size() {
			index = f.Index()
		}
	}

	return index
}

func (tfs *torrentFS) waitForMetadata() {
	for !tfs.handle.Status().GetHasMetadata() {
		time.Sleep(100 * time.Millisecond)
	}

	tfs.info = tfs.handle.TorrentFile()
}

func (tfs *torrentFS) HasTorrentInfo() bool {
	return tfs.info != nil
}

func (tfs *torrentFS) TorrentInfo() lt.TorrentInfo {
	for tfs.info == nil {
		time.Sleep(100 * time.Millisecond)
	}

	return tfs.info
}

func (tfs *torrentFS) LoadFileProgress() {
	tfs.progresses = lt.NewStd_vector_size_type()
	tfs.handle.FileProgress(tfs.progresses, int(lt.TorrentHandlePieceGranularity))
}

func (tfs *torrentFS) getFileDownloadedBytes(i int) (bytes int64) {
	defer func() {
		if res := recover(); res != nil {
			bytes = 0
		}
	}()

	bytes = tfs.progresses.Get(i)

	return
}

func (tfs *torrentFS) Files() []*torrentFile {
	info := tfs.TorrentInfo()
	files := make([]*torrentFile, info.NumFiles())

	for i := 0; i < info.NumFiles(); i++ {
		file, _ := tfs.FileAt(i)
		file.downloaded = tfs.getFileDownloadedBytes(i)
		if file.Size() > 0 {
			file.progress = float32(file.downloaded) / float32(file.Size())
		}
		files[i] = file
	}

	return files
}

func (tfs *torrentFS) SavePath() string {
	return tfs.handle.Status().GetSavePath()
}

func (tfs *torrentFS) FileAt(index int) (*torrentFile, error) {
	info := tfs.TorrentInfo()
	if index < 0 || index >= info.NumFiles() {
		return nil, errInvalidIndex
	}

	fileEntry := info.FileAt(index)
	path, _ := filepath.Abs(path.Join(tfs.SavePath(), fileEntry.GetPath()))

	return &torrentFile{
		tfs:       tfs,
		fileEntry: fileEntry,
		savePath:  path,
		index:     index,
	}, nil
}

func (tfs *torrentFS) FileByName(name string) (*torrentFile, error) {
	savePath, _ := filepath.Abs(path.Join(tfs.SavePath(), name))

	for _, file := range tfs.Files() {
		if file.SavePath() == savePath {
			return file, nil
		}
	}

	return nil, errFileNotFound
}

func (tfs *torrentFS) Open(name string) (http.File, error) {
	if tfs.shuttingDown || !tfs.HasTorrentInfo() {
		return nil, errFileNotFound
	}

	if name == "/" {
		return &torrentDir{tfs: tfs}, nil
	}

	return tfs.OpenFile(name)
}

func (tfs *torrentFS) checkPriorities() {
	for index, priority := range tfs.priorities {
		if priority == 0 {
			continue
		}

		found := false
		for _, f := range tfs.openedFiles {
			if f.index == index {
				found = true
				break
			}
		}

		if !found {
			tfs.setPriority(index, 0)
		}
	}
}

func (tfs *torrentFS) OpenFile(name string) (tf *torrentFile, err error) {
	tf, err = tfs.FileByName(name)
	if err != nil {
		return
	}

	tfs.fileCounter++
	tf.num = tfs.fileCounter

	if config.Verbose {
		tf.log("Opening %s...", tf.Name())
	}

	tf.SetPriority(1)

	startPiece, endPiece := tf.Pieces()
	if !tf.havePiece(startPiece) {
		tfs.handle.SetPieceDeadline(startPiece, 50)
	}

	if !tf.havePiece(endPiece) {
		tfs.handle.SetPieceDeadline(endPiece, 50)

		x := 0
		for i := endPiece - 6; i < endPiece; i++ {
			tfs.handle.SetPieceDeadline(i, 100+(x*50))
			x++
		}
	}

	tfs.lastOpenedFile = tf
	tfs.addOpenedFile(tf)
	tfs.checkPriorities()

	return
}

func (tf *torrentFile) SavePath() string {
	return tf.savePath
}

func (tf *torrentFile) Index() int {
	return tf.index
}

func (tf *torrentFile) Downloaded() int64 {
	return tf.downloaded
}

func (tf *torrentFile) Progress() float32 {
	return tf.progress
}

func (tf *torrentFile) FilePtr() (*os.File, error) {
	var err error
	if tf.closed {
		return nil, io.EOF
	}

	if tf.filePtr == nil {
		for {
			_, err = os.Stat(tf.savePath)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		tf.filePtr, err = os.Open(tf.savePath)
	}

	return tf.filePtr, err
}

func (tf *torrentFile) log(message string, v ...interface{}) {
	args := append([]interface{}{tf.num}, v...)
	log.Printf("T2HTTP: [%d] "+message+"\n", args...)
}

func (tf *torrentFile) Pieces() (int, int) {
	startPiece, _ := tf.pieceFromOffset(1)
	endPiece, _ := tf.pieceFromOffset(tf.Size() - 1)
	return startPiece, endPiece
}

func (tf *torrentFile) SetPriority(priority int) {
	tf.tfs.setPriority(tf.index, priority)
}

func (tf *torrentFile) Stat() (fileInfo os.FileInfo, err error) {
	return tf, nil
}

func (tf *torrentFile) readOffset() (offset int64) {
	offset, _ = tf.filePtr.Seek(0, os.SEEK_CUR)
	return
}

func (tf *torrentFile) havePiece(piece int) bool {
	return tf.tfs.handle.HavePiece(piece)
}

func (tf *torrentFile) pieceLength() int {
	return tf.tfs.info.PieceLength()
}

func (tf *torrentFile) pieceFromOffset(offset int64) (int, int) {
	pieceLength := int64(tf.tfs.info.PieceLength())
	piece := int((tf.Offset() + offset) / pieceLength)
	pieceOffset := int((tf.Offset() + offset) % pieceLength)
	return piece, pieceOffset
}

func (tf *torrentFile) Offset() int64 {
	return tf.fileEntry.GetOffset()
}

func (tf *torrentFile) waitForPiece(piece int) error {
	if tf.havePiece(piece) {
		return nil
	}

	tf.tfs.handle.SetPieceDeadline(piece, 50)

	if config.Verbose {
		tf.log("Waiting for piece %d", piece)
	}

	for !tf.havePiece(piece) {
		if tf.tfs.handle.PiecePriority(piece).(int) == 0 || tf.closed {
			return io.EOF
		}
		time.Sleep(50 * time.Millisecond)
	}

	_, endPiece := tf.Pieces()
	if piece < endPiece && !tf.havePiece(piece+1) {
		tf.tfs.handle.SetPieceDeadline(piece+1, 50)
	}

	return nil
}

func (tf *torrentFile) Close() (err error) {
	if tf.closed {
		return
	}

	if config.Verbose {
		tf.log("Closing %s...", tf.Name())
	}

	tf.tfs.removeOpenedFile(tf)
	tf.closed = true
	if tf.filePtr != nil {
		err = tf.filePtr.Close()
	}

	return
}

func (tf *torrentFile) Read(data []byte) (int, error) {
	filePtr, err := tf.FilePtr()
	if err != nil {
		return 0, err
	}

	toRead := len(data)
	if toRead > tf.pieceLength() {
		toRead = tf.pieceLength()
	}

	readOffset := tf.readOffset()
	startPiece, _ := tf.pieceFromOffset(readOffset)
	endPiece, _ := tf.pieceFromOffset(readOffset + int64(toRead))

	for i := startPiece; i <= endPiece; i++ {
		if err := tf.waitForPiece(i); err != nil {
			return 0, err
		}
	}

	tmpData := make([]byte, toRead)
	read, err := filePtr.Read(tmpData)

	if err == nil {
		copy(data, tmpData[:read])
	}

	return read, err
}

func (tf *torrentFile) Seek(offset int64, whence int) (newOffset int64, err error) {
	filePtr, err := tf.FilePtr()
	if err != nil {
		return
	}

	if whence == os.SEEK_END {
		offset = tf.Size() - offset
		whence = os.SEEK_SET
	}

	newOffset, err = filePtr.Seek(offset, whence)

	if err != nil {
		return
	}

	if config.Verbose {
		tf.log("Seeking to %d/%d", newOffset, tf.Size())
	}

	return
}

func (tf *torrentFile) Readdir(int) ([]os.FileInfo, error) {
	return make([]os.FileInfo, 0), nil
}

func (tf *torrentFile) Name() string {
	return tf.fileEntry.GetPath()
}

func (tf *torrentFile) Size() int64 {
	return tf.fileEntry.GetSize()
}

func (tf *torrentFile) Mode() os.FileMode {
	return 0
}

func (tf *torrentFile) ModTime() time.Time {
	return time.Unix(int64(tf.fileEntry.GetMtime()), 0)
}

func (tf *torrentFile) IsDir() bool {
	return false
}

func (tf *torrentFile) Sys() interface{} {
	return nil
}

func (tf *torrentFile) IsComplete() bool {
	return tf.downloaded == tf.Size()
}

func (td *torrentDir) Close() error {
	return nil
}

func (td *torrentDir) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (td *torrentDir) Readdir(count int) (files []os.FileInfo, err error) {
	info := td.tfs.TorrentInfo()
	totalFiles := info.NumFiles()
	read := &td.entriesRead
	toRead := totalFiles - *read

	if count >= 0 && count < toRead {
		toRead = count
	}

	files = make([]os.FileInfo, toRead)

	for i := 0; i < toRead; i++ {
		files[i], err = td.tfs.FileAt(*read)
		*read++
	}

	return
}

func (td *torrentDir) Seek(int64, int) (int64, error) {
	return 0, nil
}

func (td *torrentDir) Stat() (os.FileInfo, error) {
	return td, nil
}

func (td *torrentDir) Name() string {
	return "/"
}

func (td *torrentDir) Size() int64 {
	return 0
}

func (td *torrentDir) Mode() os.FileMode {
	return os.ModeDir
}

func (td *torrentDir) ModTime() time.Time {
	return time.Now()
}

func (td *torrentDir) IsDir() bool {
	return true
}

func (td *torrentDir) Sys() interface{} {
	return nil
}
