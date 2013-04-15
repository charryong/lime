package primitives

import (
	"runtime"
	"strings"
)

type (
	InnerBufferInterface interface {
		Size() int
		SubstrR(r Region) []rune
		InsertR(point int, data []rune)
		Erase(point, length int)
		Index(int) rune
		RowCol(point int) (row, col int)
		TextPoint(row, col int) (i int)
		Close()
	}
	Buffer interface {
		InnerBufferInterface
		IdInterface
		SettingsInterface
		AddCallback(cb BufferChangedCallback)
		SetName(string)
		Name() string
		SetFileName(string)
		FileName() string
		Insert(point int, svalue string)
		Substr(r Region) string
		String() string
		Runes() []rune
		ChangeCount() int
		Line(offset int) Region
		Lines(r Region) Region
		FullLine(offset int) Region
		FullLines(r Region) Region
		Word(offset int) Region
		Words(r Region) Region
	}
	BufferChangedCallback func(buf Buffer, position, delta int)

	buffer struct {
		HasId
		HasSettings
		SerializedBuffer
		changecount int
		name        string
		filename    string
		callbacks   []BufferChangedCallback
	}
)

func NewBuffer() Buffer {
	b := buffer{}
	b.SerializedBuffer.init(&node{})
	r := &b
	runtime.SetFinalizer(r, func(b *buffer) { b.Close() })

	return r
}

func (b *buffer) AddCallback(cb BufferChangedCallback) {
	b.callbacks = append(b.callbacks, cb)
}

func (b *buffer) SetName(n string) {
	b.name = n
}

func (b *buffer) Name() string {
	return b.name
}

func (b *buffer) FileName() string {
	return b.filename
}

func (b *buffer) SetFileName(n string) {
	b.filename = n
}

func (buf *buffer) notify(position, delta int) {
	for i := range buf.callbacks {
		buf.callbacks[i](buf, position, delta)
	}
}

func (buf *buffer) Insert(point int, svalue string) {
	if len(svalue) == 0 {
		return
	}
	value := []rune(svalue)
	buf.SerializedBuffer.InsertR(point, value)
	buf.changecount++
	buf.notify(point, len(value))
}

func (buf *buffer) Erase(point, length int) {
	if length == 0 {
		return
	}
	buf.changecount++
	buf.SerializedBuffer.Erase(point, length)
	buf.notify(point+length, -length)
}

func (b *buffer) Substr(r Region) string {
	return string(b.SubstrR(r))
}

func (b *buffer) String() string {
	return b.Substr(Region{0, b.Size()})
}

func (b *buffer) Runes() []rune {
	return b.SubstrR(Region{0, b.Size()})
}

func (b *buffer) ChangeCount() int {
	return b.changecount
}

func (b *buffer) Line(offset int) Region {
	if offset < 0 {
		return Region{0, 0}
	} else if s := b.Size(); offset >= s {
		return Region{s, s}
	}
	soffset := offset
sloop:
	o := Clamp(0, soffset, soffset-32)
	sub := b.SubstrR(Region{o, soffset})
	s := soffset
	for s > o && sub[s-o-1] != '\n' {
		s--
	}
	if s == o && o > 0 && sub[0] != '\n' {
		soffset = o
		goto sloop
	}

	l := b.Size()
	eoffset := offset
eloop:
	o = Clamp(eoffset, l, eoffset+32)
	sub = b.SubstrR(Region{eoffset, o})
	e := eoffset
	for e < o && sub[e-eoffset] != '\n' {
		e++
	}
	if e == o && o < l && sub[o-eoffset-1] != '\n' {
		eoffset = o
		goto eloop
	}
	return Region{s, e}
}

// Returns a region that starts at the first character in a line
// and ends with the last character in a (possibly different) line
func (b *buffer) Lines(r Region) Region {
	s := b.Line(r.Begin())
	e := b.Line(r.End())
	return Region{s.Begin(), e.End()}
}

func (b *buffer) FullLine(offset int) Region {
	r := b.Line(offset)
	s := b.Size()
	for r.B < s {
		if i := b.Index(r.B); i == '\r' || i == '\n' {
			break
		}
		r.B++
	}
	if r.B != b.Size() {
		r.B++
	}
	return r
}

// Returns a region that starts at the first character in a line
// and ends with the line break in a (possibly different) line
func (b *buffer) FullLines(r Region) Region {
	s := b.FullLine(r.Begin())
	e := b.FullLine(r.End())
	return Region{s.Begin(), e.End()}
}

func (b *buffer) Word(offset int) Region {
	if offset < 0 {
		offset = 0
	}
	lr := b.FullLine(offset)
	col := offset - lr.Begin()

	line := b.SubstrR(lr)
	if len(line) == 0 {
		return Region{offset, offset}
	}

	seps := "./\\()\"'-:,.;<>~!@#$%^&*|+=[]{}`~?"
	if v, ok := b.Settings().Get("word_separators", seps).(string); ok {
		seps = v
	}
	spacing := " \n\t\r"
	eseps := seps + spacing

	if col >= len(line) {
		col = len(line) - 1
	}
	last := true
	li := 0
	ls := false
	lc := 0
	for i, r := range line {
		cur := strings.ContainsRune(eseps, r)
		cs := r == ' '
		if !cs {
			lc = i
		}
		if last != cur || ls != cs {
			ls = cs
			r := Region{li, i}
			if r.Contains(col) && i != 0 {
				r.A, r.B = r.A+lr.Begin(), r.B+lr.Begin()
				if !(r.B == offset && last) {
					return r
				}
			}
			li = i
			last = cur
		}
	}
	r := Region{lr.Begin() + li, lr.End()}
	lc += lr.Begin()
	if lc != offset && !strings.ContainsRune(spacing, b.Index(r.A)) {
		r.B = lc
	}
	if r.A == offset && r.B == r.A+1 {
		r.B--
	}
	return r
}

// Returns a region that starts at the first character in a word
// and ends with the last character in a (possibly different) word
func (b *buffer) Words(r Region) Region {
	s := b.Word(r.Begin())
	e := b.Word(r.End())
	return Region{s.Begin(), e.End()}
}
