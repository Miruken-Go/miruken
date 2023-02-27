package api

import (
	"crypto/rand"
	"fmt"
	"github.com/miruken-go/miruken"
	"github.com/miruken-go/miruken/maps"
	"io"
	"mime"
	"strings"
)

type (
	// Part represents a piece of a message.
	// Part can also be used to provide explicit message details.
	Part interface {
		ContentType() string
		Metadata() map[string][]string
		Filename() string
		Content() any
	}

	// PartContainer stores all the Part's of a message.
	// The main part is typically the message payload.
	PartContainer interface {
		ContentType() string
		Metadata() map[string][]string
		Boundary() string
		Keys() []string
		MainKey() string
		Part(key string) Part
	}

	// PartBuilder builds a message Part.
	PartBuilder struct {
		part partEntry
	}

	// ReadPartsBuilder builds a PartContainer for reading Part's.
	ReadPartsBuilder struct {
		container partContainer
		firstKey  string
	}

	// WritePartsBuilder builds a PartContainer for writing Part's.
	WritePartsBuilder struct {
		ReadPartsBuilder
	}

	partEntry struct {
		contentType string
		metadata    map[string][]string
		filename    string
		content     any
	}

	partContainer struct {
		contentType string
		boundary    string
		metadata    map[string][]string
		parts       map[string]Part
		main        string
	}
)


// PartBuilder

func (b *PartBuilder) ContentType(
	contentType string,
) *PartBuilder {
	if _, _, err := mime.ParseMediaType(contentType); err != nil {
		panic(fmt.Sprintf("invalid content type: %q", contentType))
	}
	b.part.contentType = contentType
	return b
}

func (b *PartBuilder) Metadata(
	metadata map[string][]string,
) *PartBuilder {
	for key, val := range metadata {
		m := b.part.metadata
		if m == nil {
			m = make(map[string][]string)
			b.part.metadata = m
		}
		if v, ok := m[key]; ok {
			m[key] = append(v, val...)
		} else {
			m[key] = val
		}
	}
	return b
}

func (b *PartBuilder) Filename(
	filename string,
) *PartBuilder {
	b.part.filename = filename
	return b
}

func (b *PartBuilder) Content(
	content any,
) *PartBuilder {
	if miruken.IsNil(content) {
		panic("content cannot be nil")
	}
	b.part.content = content
	return b
}

func (b *PartBuilder) Build() Part {
	return b.part
}


// ReadPartsBuilder

func (b *ReadPartsBuilder) MainKey(
	key string,
) *ReadPartsBuilder {
	b.container.main = key
	return b
}

func (b *ReadPartsBuilder) AddPart(
	key  string,
	part Part,
) *ReadPartsBuilder {
	if len(key) == 0 {
		panic("key cannot be empty")
	}
	if miruken.IsNil(part) {
		panic("part cannot be nil")
	}
	parts := b.container.parts
	if parts == nil {
		parts = make(map[string]Part)
		b.container.parts = parts
	} else if _, ok := parts[key]; ok {
		panic(fmt.Sprintf("part with key %q already added", key))
	}
	parts[key] = part
	if len(b.firstKey) == 0 {
		b.firstKey = key
	}
	return b
}

func (b *ReadPartsBuilder) Build() PartContainer {
	if len(b.container.main) == 0 {
		b.container.main = b.firstKey
	}
	return b.container
}


// WritePartsBuilder

func (b *WritePartsBuilder) ContentType(
	contentType string,
) *WritePartsBuilder {
	if _, params, err := mime.ParseMediaType(contentType); err != nil {
		panic(fmt.Sprintf("invalid content type: %q", contentType))
	} else {
		b.container.contentType = contentType
		b.container.boundary    = params["boundary"]
	}
	return b
}

func (b *WritePartsBuilder) Boundary(
	boundary string,
) *WritePartsBuilder {
	b.container.boundary = boundary
	return b
}

func (b *WritePartsBuilder) Build() PartContainer {
	ctr := &b.container
	if len(ctr.main) == 0 {
		ctr.main = b.firstKey
	}
	if boundary := ctr.boundary; len(boundary) == 0 {
		ctr.boundary = randomBoundary()
	} else if strings.ContainsAny(boundary, `()<>@,;:\"/[]?= `) {
		ctr.boundary = `"` + boundary + `"`
	}
	if len(ctr.contentType) == 0 {
		ctr.contentType = "multipart/form-data"
	} else if !strings.HasPrefix(ctr.contentType, "multipart/") {
		ctr.contentType = "multipart/" + ctr.contentType
	}
	if !strings.Contains(ctr.contentType, "boundary") {
		ctr.contentType = ctr.contentType + "; boundary=" + ctr.boundary
	}
	return ctr
}


// partEntry

func (p partEntry) ContentType() string {
	return p.contentType
}

func (p partEntry) Metadata() map[string][]string {
	return p.metadata
}

func (p partEntry) Filename() string {
	return p.filename
}

func (p partEntry) Content() any {
	return p.content
}


// partContainer

func (c partContainer) ContentType() string {
	return c.contentType
}

func (c partContainer) Boundary() string {
	return c.boundary
}

func (c partContainer) Metadata() map[string][]string {
	return c.metadata
}

func (c partContainer) Keys() []string {
	return maps.Keys(c.parts)
}

func (c partContainer) MainKey() string {
	return c.main
}

func (c partContainer) Part(key string) Part {
	if parts := c.parts; parts != nil {
		if p, ok := parts[key]; ok {
			return p
		}
	}
	return nil
}


// randomBoundary copied from multipart.randomBoundary
func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}
