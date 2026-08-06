package scanner

import (
	"Disko/internal/tree"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type Progress struct {
	Count int64
	Path  string
}

type job struct {
	path   string
	parent *tree.Entry
	wg     *sync.WaitGroup
}

type Scanner struct {
	PChan chan Progress
	count int64
	jobs  chan job
}

func New() *Scanner {
	return &Scanner{
		PChan: make(chan Progress, 256),
		jobs:  make(chan job, 2048),
	}
}

func (s *Scanner) Scan(root string) (*tree.Tree, error) {
	fi, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	rootNode := &tree.Entry{
		Type: tree.Folder,
		Name: fi.Name(),
	}
	if root == "/" {
		rootNode.Name = "/"
	}

	for i := 0; i < 128; i++ {
		go s.worker()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	s.jobs <- job{path: root, parent: rootNode, wg: &wg}

	wg.Wait()
	close(s.jobs)
	close(s.PChan)

	tree.CalcSizes(rootNode)

	return &tree.Tree{Root: rootNode}, nil
}

func (s *Scanner) worker() {
	for j := range s.jobs {
		s.process(j)
	}
}

func (s *Scanner) process(j job) {
	defer j.wg.Done()

	ents, err := os.ReadDir(j.path)
	if err != nil {
		return
	}

	kids := make([]*tree.Entry, 0, len(ents))

	for _, e := range ents {
		n := atomic.AddInt64(&s.count, 1)

		if n%300 == 0 {
			select {
			case s.PChan <- Progress{Count: n, Path: j.path}:
			default:
			}
		}

		name := e.Name()
		if j.path == "/" && (name == "proc" || name == "sys" || name == "dev") {
			continue
		}

		node := &tree.Entry{
			Name: name,
			Type: tree.File,
		}

		if e.IsDir() {
			node.Type = tree.Folder
			j.wg.Add(1)

			childJob := job{path: filepath.Join(j.path, name), parent: node, wg: j.wg}

			select {
			case s.jobs <- childJob:
			default:
				go func(cj job) {

					s.jobs <- cj
				}(childJob)
			}
		} else {
			if info, err := e.Info(); err == nil {
				node.Size = info.Size()
				if info.Mode()&os.ModeSymlink != 0 {
					node.Type = tree.Symlink
				}
			}
		}
		kids = append(kids, node)
	}

	j.parent.Children = kids
}
