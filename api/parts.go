package api

import (
	"crypto/rand"
	"fmt"
	"github.com/miruken-go/miruken"
	"io"
	"mime"
	"strings"
)

type (
	// Part represents a piece of a message.
	// Part can also be used to provide explicit message details.
	Part interface {
		Content
		Filename() string
	}

	// PartContainer stores all the Part's of a message.
	// The main part is typically the message payload.
	PartContainer interface {
		Content
		Parts() map[string]Part
		MainPart() Part
	}

	// PartBuilder builds a message Part.
	PartBuilder struct {
		part part
	}

	// ReadPartsBuilder builds a PartContainer for reading Part's.
	ReadPartsBuilder struct {
		container partContainer
	}

	// WritePartsBuilder builds a PartContainer for writing Part's.
	WritePartsBuilder struct {
		ReadPartsBuilder
	}

	part struct {
		mediaType string
		metadata  map[string][]any
		filename  string
		body      any
	}

	partContainer struct {
		mediaType string
		boundary  string
		metadata  map[string][]any
		parts     map[string]Part
		main      Part
	}
)


// PartBuilder

func (b *PartBuilder) MediaType(
	mediaType string,
) *PartBuilder {
	if _, _, err := mime.ParseMediaType(mediaType); err != nil {
		panic(fmt.Errorf("invalid media type %q (%w)", mediaType, err))
	}
	b.part.mediaType = mediaType
	return b
}

func (b *PartBuilder) Metadata(
	metadata map[string][]any,
) *PartBuilder {
	for key, val := range metadata {
		m := b.part.metadata
		if m == nil {
			m = make(map[string][]any)
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

func (b *PartBuilder) MetadataStrings(
	metadata map[string][]string,
) *PartBuilder {
	meta := make(map[string][]any)
	for k, arr := range metadata {
		s := make([]any, len(arr))
		for i, v := range arr {
			s[i] = v
		}
		meta[k] = s
	}
	return b
}

func (b *PartBuilder) Filename(
	filename string,
) *PartBuilder {
	b.part.filename = filename
	return b
}

func (b *PartBuilder) Body(
	body any,
) *PartBuilder {
	if miruken.IsNil(body) {
		panic("body cannot be nil")
	}
	b.part.body = body
	return b
}

func (b *PartBuilder) Build() Part {
	part := b.part
	if part.metadata == nil {
		part.metadata = make(map[string][]any)
	}
	return part
}


// ReadPartsBuilder

func (b *ReadPartsBuilder) Metadata(
	metadata map[string][]any,
) *ReadPartsBuilder {
	for key, val := range metadata {
		m := b.container.metadata
		if m == nil {
			m = make(map[string][]any)
			b.container.metadata = m
		}
		if v, ok := m[key]; ok {
			m[key] = append(v, val...)
		} else {
			m[key] = val
		}
	}
	return b
}

func (b *ReadPartsBuilder) MainPart(
	main Part,
) *ReadPartsBuilder {
	b.container.main = main
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
	return b
}

func (b *ReadPartsBuilder) AddParts(
	parts map[string]Part,
) *ReadPartsBuilder {
	for key, part := range parts {
		b.AddPart(key, part)
	}
	return b
}

func (b *ReadPartsBuilder) NewPart() *PartBuilder {
	return &PartBuilder{}
}

func (b *ReadPartsBuilder) Build() PartContainer {
	ctr := b.container
	if ctr.parts == nil {
		ctr.parts = make(map[string]Part)
	}
	if ctr.metadata == nil {
		ctr.metadata = make(map[string][]any)
	}
	return ctr
}


// WritePartsBuilder

func (b *WritePartsBuilder) MediaType(
	mediaType string,
) *WritePartsBuilder {
	if _, params, err := mime.ParseMediaType(mediaType); err != nil {
		panic(fmt.Errorf("invalid media type %q: %w", mediaType, err))
	} else {
		b.container.mediaType = mediaType
		b.container.boundary    = params["boundary"]
	}
	return b
}

func (b *WritePartsBuilder) Build() PartContainer {
	ctr := &b.container
	if ctr.parts == nil {
		ctr.parts = make(map[string]Part)
	}
	if ctr.metadata == nil {
		ctr.metadata = make(map[string][]any)
	}
	if len(ctr.mediaType) == 0 {
		ctr.mediaType = "multipart/form-data"
	} else if !strings.HasPrefix(ctr.mediaType, "multipart/") {
		ctr.mediaType = "multipart/" + ctr.mediaType
	}
	if len(ctr.boundary) == 0 {
		ctr.mediaType = ctr.mediaType + "; boundary=" + randomBoundary()
	}
	return ctr
}


// part

func (p part) MediaType() string {
	return p.mediaType
}

func (p part) Metadata() map[string][]any {
	return p.metadata
}

func (p part) Filename() string {
	return p.filename
}

func (p part) Body() any {
	return p.body
}


// partContainer

func (c partContainer) MediaType() string {
	return c.mediaType
}

func (c partContainer) Metadata() map[string][]any {
	return c.metadata
}

func (c partContainer) Parts() map[string]Part {
	return c.parts
}

func (c partContainer) MainPart() Part {
	return c.main
}

func (c partContainer) Body() any {
	return c
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
