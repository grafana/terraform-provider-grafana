package generator

import (
	"io"
	"io/fs"
	"os"
)

type RepoRootFs interface {
	fs.FS
	Root() string
}

type RepoRootDirFs struct {
	fs.FS
	root string
}

func (r *RepoRootDirFs) Root() string {
	return r.root
}

type Generator struct {
	fs     RepoRootFs
	writer io.Writer
}

func New(repoRoot string, writer io.Writer) *Generator {
	return &Generator{
		fs:     &RepoRootDirFs{FS: os.DirFS(repoRoot), root: repoRoot},
		writer: writer,
	}
}

func (g *Generator) Generate(pathsToCheck []string) error {
	return nil
}
