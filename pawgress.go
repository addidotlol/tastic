package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	pawgressSlots    = 20
	pawgressNameSize = 40
	pawPrint         = "🐾"
	emptySlot        = "░░"
)

type pawgress struct {
	mu      sync.Mutex
	out     io.Writer
	visible bool
	label   string
	total   int
	done    int
}

func newPawgress(label string, total int) *pawgress {
	p := &pawgress{out: os.Stderr, visible: isTerminal(os.Stderr), label: label, total: total}
	p.draw("")
	return p
}

func (p *pawgress) step(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	p.draw(name)
}

func (p *pawgress) finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.visible {
		fmt.Fprint(p.out, "\r\033[K")
	}
}

func (p *pawgress) draw(name string) {
	if !p.visible {
		return
	}
	filled := pawgressSlots
	if p.total > 0 {
		filled = p.done * pawgressSlots / p.total
	}
	bar := strings.Repeat(pawPrint, filled) + strings.Repeat(emptySlot, pawgressSlots-filled)
	fmt.Fprintf(p.out, "\r\033[K%s %s %d/%d %s", p.label, bar, p.done, p.total, shortenFromLeft(name, pawgressNameSize))
}

func shortenFromLeft(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return "…" + string(runes[len(runes)-max+1:])
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
