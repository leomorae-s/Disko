package tree

type EntryType uint8

const (
	_ EntryType = iota
	File
	Folder
	Symlink
)

type Tree struct {
	Root *Entry
}

type Entry struct {
	Name     string
	Children []*Entry
	Size     int64
	Type     EntryType
}

func CalcSizes(e *Entry) int64 {
	if e == nil {
		return 0
	}
	if e.Type != Folder {
		return e.Size
	}

	var tot int64
	for _, c := range e.Children {
		tot += CalcSizes(c)
	}
	e.Size = tot
	return tot
}
