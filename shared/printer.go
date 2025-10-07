package shared

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

type StringWriteCloser interface {
	io.Closer
	io.StringWriter
}

type WriteCloser struct {
	w io.WriteCloser
}

func NewWriteCloser(w io.WriteCloser) StringWriteCloser {
	if w == nil {
		return nil
	}
	return &WriteCloser{w: w}
}

func (wc *WriteCloser) WriteString(s string) (n int, err error) {
	return wc.w.Write([]byte(s))
}

func (wc *WriteCloser) Close() error {
	return wc.w.Close()
}

type Printer struct {
	mu     sync.Mutex
	indStr string
	hooks  []StringWriteCloser
}

func NewPrinter(indentString string, hooks ...StringWriteCloser) (*Printer, error) {
	p := &Printer{
		indStr: indentString,
	}
	if len(hooks) == 0 {
		return nil, errors.New("no hook provided")
	}
	for _, hook := range hooks {
		if hook == nil {
			return nil, errors.New("a nil pointed hook is given")
		}
	}
	p.hooks = hooks
	return p, nil
}

func (p *Printer) Write(s string, ind int) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	indent := strings.Repeat(p.indStr, ind)
	firstLine := true
	for line := range strings.SplitSeq(s, "\n") {
		if !firstLine {
			line = "\n" + indent + line
		} else {
			firstLine = false
			line = indent + line
		}
		for _, hook := range p.hooks {
			if _, err := hook.WriteString(line); err != nil {
				return fmt.Errorf("on writing to hook: %w", err)
			}
		}
	}
	return nil
}

func (p *Printer) Writeln(s string, ind int) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	indent := strings.Repeat(p.indStr, ind)
	firstLine := true
	for line := range strings.SplitSeq(s, "\n") {
		if !firstLine {
			line = "\n" + indent + line
		} else {
			firstLine = false
			line = indent + line
		}
		for _, hook := range p.hooks {
			if _, err := hook.WriteString(line); err != nil {
				return fmt.Errorf("on writing to hook: %w", err)
			}
		}
	}
	for _, hook := range p.hooks {
		if _, err := hook.WriteString("\n"); err != nil {
			return fmt.Errorf("on writing newline to hook: %w", err)
		}
	}
	return nil
}

func (p *Printer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, hook := range p.hooks {
		if err := hook.Close(); err != nil {
			return fmt.Errorf("on closing hook: %w", err)
		}
	}
	return nil
}
