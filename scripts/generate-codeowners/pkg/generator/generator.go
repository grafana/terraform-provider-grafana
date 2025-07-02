package generator

import (
	"encoding/json"
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
	fs               RepoRootFs
	codeownersWriter io.Writer
	ownershipReader  io.Reader
}

func New(repoRoot string, codeownersFile io.Writer, ownershipFile io.Reader) *Generator {
	return &Generator{
		fs:               &RepoRootDirFs{FS: os.DirFS(repoRoot), root: repoRoot},
		codeownersWriter: codeownersFile,
		ownershipReader:  ownershipFile,
	}
}

func (g *Generator) Generate(pathsToCheck []string) error {
	g.readOwnershipFile()
	return nil
}

func (g *Generator) readOwnershipFile() error {
	ownershipFile, err := io.ReadAll(g.ownershipReader)
	if err != nil {
		return err
	}

	var ownership []map[string]interface{}
	if err := json.Unmarshal(ownershipFile, &ownership); err != nil {
		return err
	}

	return nil
}
