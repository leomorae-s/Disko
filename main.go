package main

import (
	"Disko/internal/scanner"
	"Disko/internal/tree"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	stTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	stItem  = lipgloss.NewStyle().PaddingLeft(2)
	stSel   = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	stMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func fmtSize(b int64) string {
	const u = 1024
	if b < u {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(u), 0
	for n := b / u; n >= u; n /= u {
		div *= u
		exp++
	}
	switch exp {
	case 0:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(div))
	case 1:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(div))
	case 2:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(div))
	case 3:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(div))
	default:
		return fmt.Sprintf("%.1f PB", float64(b)/float64(div))
	}
}

func sortBySz(e *tree.Entry) {
	if e == nil {
		return
	}
	sort.Slice(e.Children, func(i, j int) bool {
		return e.Children[i].Size > e.Children[j].Size
	})
	for _, c := range e.Children {
		sortBySz(c)
	}
}

type breadcrumb struct {
	node *tree.Entry
	idx  int
}

type app struct {
	curr    *tree.Entry
	history []breadcrumb
	cursor  int
	done    bool
	err     error
	root    string
	w, h    int

	sc      *scanner.Scanner
	scanned int64
	curPath string
}

type scanDone struct {
	t   *tree.Tree
	err error
}

type progMsg scanner.Progress

func newApp(dir string) app {
	return app{
		root: dir,
		sc:   scanner.New(),
	}
}

func (a app) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			t, err := a.sc.Scan(a.root)
			if t != nil && t.Root != nil {
				sortBySz(t.Root)
			}
			return scanDone{t: t, err: err}
		},
		a.tick(),
	)
}

func (a app) tick() tea.Cmd {
	return func() tea.Msg {
		if p, ok := <-a.sc.PChan; ok {
			return progMsg(p)
		}
		return nil
	}
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		return a, nil

	case progMsg:
		a.scanned = msg.Count
		a.curPath = msg.Path
		return a, a.tick()

	case scanDone:
		if msg.err != nil || msg.t == nil || msg.t.Root == nil {
			a.err = fmt.Errorf("scan failed: %v", msg.err)
			return a, tea.Quit
		}
		a.curr = msg.t.Root
		a.done = true
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "up", "k":
			if a.cursor > 0 {
				a.cursor--
			}
		case "down", "j":
			if a.done && a.curr != nil && a.cursor < len(a.curr.Children)-1 {
				a.cursor++
			}
		case "enter", "right", "l":
			if a.done && a.curr != nil && len(a.curr.Children) > 0 {
				sel := a.curr.Children[a.cursor]
				if sel.Type == tree.Folder {
					a.history = append(a.history, breadcrumb{a.curr, a.cursor})
					a.curr = sel
					a.cursor = 0
				}
			}
		case "esc", "left", "h", "backspace":
			if n := len(a.history); n > 0 {
				last := a.history[n-1]
				a.curr = last.node
				a.cursor = last.idx
				a.history = a.history[:n-1]
			}
		}
	}
	return a, nil
}

func (a app) View() string {
	if a.err != nil {
		return fmt.Sprintf("\nFatal: %v\n", a.err)
	}

	if !a.done {
		spin := "⏳"
		if a.scanned%2 == 0 {
			spin = "⌛"
		}
		p := a.curPath
		if len(p) > 60 {
			p = "..." + p[len(p)-57:]
		}
		return fmt.Sprintf("\n  %s reading: %s\n\n  found: %s",
			spin,
			lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(p),
			lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", a.scanned)),
		)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(stTitle.Render(fmt.Sprintf(" Disko 💿 - %s (%s)", a.curr.Name, fmtSize(a.curr.Size))))
	b.WriteString("\n\n")

	viewH := a.h - 6
	if viewH < 1 {
		viewH = 1
	}

	start := 0
	if a.cursor > viewH/2 {
		start = a.cursor - viewH/2
	}
	end := start + viewH
	if end > len(a.curr.Children) {
		end = len(a.curr.Children)
	}

	for i := start; i < end; i++ {
		c := a.curr.Children[i]
		ico := "📄"
		if c.Type == tree.Folder {
			ico = "📁"
		} else if c.Type == tree.Symlink {
			ico = "🔗"
		}

		ptr := " "
		if a.cursor == i {
			ptr = ">"
		}

		ln := fmt.Sprintf("%s %s %s [%s]", ptr, ico, c.Name, fmtSize(c.Size))
		if a.cursor == i {
			b.WriteString(stSel.Render(ln) + "\n")
		} else {
			b.WriteString(stItem.Render(ln) + "\n")
		}
	}

	if end < len(a.curr.Children) {
		b.WriteString(stItem.Render(fmt.Sprintf("  ... %d more ...", len(a.curr.Children)-end)) + "\n")
	}

	b.WriteString(stMuted.Render("\n  [↑/↓, j/k]: nav • [Enter, →, l]: open • [Esc, ←, h]: back • [q]: quit\n"))
	return b.String()
}

func main() {
	dir := "/"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else if h, err := os.UserHomeDir(); err == nil {
		dir = h
	}

	if _, err := tea.NewProgram(newApp(dir), tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("err:", err)
		os.Exit(1)
	}
}
