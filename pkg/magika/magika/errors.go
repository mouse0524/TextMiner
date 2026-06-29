package magika

import "errors"

var (
	ErrScannerNotInitialized = errors.New("magika scanner not initialized, call InitScanner first")
)
