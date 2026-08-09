package fileresponder

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/amalshaji/portr/internal/client/localserver"
)

type Route struct {
	Subdomain string
	Dir       string
}

// Responder serves static directories over the shared local server.
type Responder struct {
	*localserver.Server
}

func New() *Responder {
	return &Responder{Server: localserver.New("static responder")}
}

func (r *Responder) Register(rt Route) error {
	if rt.Dir == "" {
		return fmt.Errorf("dir is required")
	}

	handler, err := newDirHandler(rt.Dir)
	if err != nil {
		return err
	}
	if err := r.Server.Register(rt.Subdomain, handler); err != nil {
		_ = handler.Close()
		return err
	}
	return nil
}

// dirHandler serves one directory. It holds the open root so the server can
// close it when the route goes away.
type dirHandler struct {
	root *os.Root
	http.Handler
}

func (d *dirHandler) Close() error {
	return d.root.Close()
}

func newDirHandler(dir string) (*dirHandler, error) {
	// os.Root keeps symlinks from escaping the served directory, which http.Dir
	// explicitly does not do. The directory is published to the internet.
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to serve dir %q: %w", dir, err)
	}
	return &dirHandler{
		root:    root,
		Handler: http.FileServerFS(containedFS{root.FS()}),
	}, nil
}

// containedFS reports every unreadable path as missing. os.Root rejects
// symlinks that escape the served directory with a "path escapes from parent"
// error, which http.FileServerFS would otherwise surface as a 500 — telling a
// public client that the path exists and points somewhere it should not.
type containedFS struct {
	fsys fs.FS
}

func (c containedFS) Open(name string) (fs.File, error) {
	file, err := c.fsys.Open(name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return file, err
}
