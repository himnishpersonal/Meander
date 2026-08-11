package product

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type LocalObjects struct{ Root string }

func (o LocalObjects) path(key string) (string, error) {
	clean := filepath.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", errors.New("unsafe object key")
	}
	return filepath.Join(o.Root, clean), nil
}
func (o LocalObjects) Put(_ context.Context, key, _ string, value []byte) error {
	path, err := o.path(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	return os.WriteFile(path, value, 0640)
}
func (o LocalObjects) Get(_ context.Context, key string) ([]byte, string, error) {
	path, err := o.path(key)
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return b, contentType(key), nil
}
func (o LocalObjects) Delete(_ context.Context, key string) error {
	path, err := o.path(key)
	if err != nil {
		return err
	}
	if err = os.Remove(path); os.IsNotExist(err) {
		return nil
	}
	return err
}
func contentType(key string) string {
	switch filepath.Ext(key) {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
