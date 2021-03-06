// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func newMetadataVorbis() *metadataVorbis {
	return &metadataVorbis{
		c: make(map[string]string),
	}
}

type metadataVorbis struct {
	c map[string]string // the vorbis comments
	p *Picture
}

func (m *metadataVorbis) readVorbisComment(r io.Reader) error {
	vendorLen, err := readInt32LittleEndian(r)
	if err != nil {
		return err
	}

	vendor, err := readString(r, vendorLen)
	if err != nil {
		return err
	}
	m.c["vendor"] = vendor

	commentsLen, err := readInt32LittleEndian(r)
	if err != nil {
		return err
	}

	for i := 0; i < commentsLen; i++ {
		l, err := readInt32LittleEndian(r)
		if err != nil {
			return err
		}
		s, err := readString(r, l)
		if err != nil {
			return err
		}
		k, v, err := parseComment(s)
		if err != nil {
			return err
		}
		m.c[strings.ToLower(k)] = v
	}
	return nil
}

func (m *metadataVorbis) readPictureBlock(r io.Reader) error {
	b, err := readInt(r, 4)
	if err != nil {
		return err
	}
	pictureType, ok := pictureTypes[byte(b)]
	if !ok {
		return fmt.Errorf("invalid picture type: %v", b)
	}
	mimeLen, err := readInt(r, 4)
	if err != nil {
		return err
	}
	mime, err := readString(r, mimeLen)
	if err != nil {
		return err
	}

	ext := ""
	switch mime {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/gif":
		ext = "gif"
	}

	descLen, err := readInt(r, 4)
	if err != nil {
		return err
	}
	desc, err := readString(r, descLen)
	if err != nil {
		return err
	}

	// We skip width <32>, height <32>, colorDepth <32>, coloresUsed <32>
	_, err = readInt(r, 4) // width
	if err != nil {
		return err
	}
	_, err = readInt(r, 4) // height
	if err != nil {
		return err
	}
	_, err = readInt(r, 4) // color depth
	if err != nil {
		return err
	}
	_, err = readInt(r, 4) // colors used
	if err != nil {
		return err
	}

	dataLen, err := readInt(r, 4)
	if err != nil {
		return err
	}
	data := make([]byte, dataLen)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return err
	}

	m.p = &Picture{
		Ext:         ext,
		MIMEType:    mime,
		Type:        pictureType,
		Description: desc,
		Data:        data,
	}
	return nil
}

func parseComment(c string) (k, v string, err error) {
	kv := strings.SplitN(c, "=", 2)
	if len(kv) != 2 {
		err = errors.New("vorbis comment must contain '='")
		return
	}
	k = kv[0]
	v = kv[1]
	return
}

func (m *metadataVorbis) Format() Format {
	return VORBIS
}

func (m *metadataVorbis) Raw() map[string]interface{} {
	raw := make(map[string]interface{}, len(m.c))
	for k, v := range m.c {
		raw[k] = v
	}
	return raw
}

func (m *metadataVorbis) Title() string {
	return m.c["title"]
}

func (m *metadataVorbis) Artist() string {
	// PERFORMER
	// The artist(s) who performed the work. In classical music this would be the
	// conductor, orchestra, soloists. In an audio book it would be the actor who
	// did the reading. In popular music this is typically the same as the ARTIST
	// and is omitted.
	if m.c["performer"] != "" {
		return m.c["performer"]
	}
	return m.c["artist"]
}

func (m *metadataVorbis) Album() string {
	return m.c["album"]
}

func (m *metadataVorbis) AlbumArtist() string {
	// This field isn't actually included in the standard, though
	// it is commonly assigned to albumartist.
	return m.c["albumartist"]
}

func (m *metadataVorbis) Composer() string {
	// ARTIST
	// The artist generally considered responsible for the work. In popular music
	// this is usually the performing band or singer. For classical music it would
	// be the composer. For an audio book it would be the author of the original text.
	if m.c["composer"] != "" {
		return m.c["composer"]
	}
	if m.c["performer"] == "" {
		return ""
	}
	return m.c["artist"]
}

func (m *metadataVorbis) Genre() string {
	return m.c["genre"]
}

func (m *metadataVorbis) Year() int {
	// FIXME: try to parse the date in m.c["date"] to extract this
	return 0
}

func (m *metadataVorbis) Track() (int, int) {
	x, _ := strconv.Atoi(m.c["tracknumber"])
	// https://wiki.xiph.org/Field_names
	n, _ := strconv.Atoi(m.c["tracktotal"])
	return x, n
}

func (m *metadataVorbis) Disc() (int, int) {
	// https://wiki.xiph.org/Field_names
	x, _ := strconv.Atoi(m.c["discnumber"])
	n, _ := strconv.Atoi(m.c["disctotal"])
	return x, n
}

func (m *metadataVorbis) Lyrics() string {
	return m.c["lyrics"]
}

func (m *metadataVorbis) Picture() *Picture {
	return m.p
}
