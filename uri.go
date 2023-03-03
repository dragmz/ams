package ams

import (
	"github.com/dragmz/wc"
	"github.com/pkg/errors"
)

type UriSource struct {
	uri     string
	cb      bool
	require bool
}

type UriSourceOption func(s *UriSource)

// WithUriSourceNonEmpty set to true makes the UriSource return an error when reading uri and no sources are provided
func WithUriSourceNonEmpty(requireNonEmpty bool) UriSourceOption {
	return func(s *UriSource) {
		s.require = requireNonEmpty
	}
}

func WithUriSourceStaticUri(uri string) UriSourceOption {
	return func(s *UriSource) {
		s.uri = uri
	}
}

func WithUriSourceClipboardUri(enable bool) UriSourceOption {
	return func(s *UriSource) {
		s.cb = enable
	}
}

func MakeUriSource(opts ...UriSourceOption) (*UriSource, error) {
	s := &UriSource{}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *UriSource) Uri() (*wc.Uri, error) {
	uris := 0

	if len(s.uri) > 0 {
		uris++
	}

	if s.cb {
		uris++
	}

	if uris > 1 {
		return nil, errors.New("cannot read uri from multiple sources")
	}

	if len(s.uri) > 0 {
		return wc.ParseUri(s.uri)
	}

	if s.cb {
		uri, err := ReadWcFromClipboard()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read uri from clipboard")
		}

		return wc.ParseUri(uri)
	}

	if s.require {
		return nil, errors.New("missing uri source")
	}

	return nil, nil
}
