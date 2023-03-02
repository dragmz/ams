package ams

import "github.com/pkg/errors"

type UriSource struct {
	uri string
	cb  bool
}

type UriSourceOption func(s *UriSource)

func WithStaticUri(uri string) UriSourceOption {
	return func(s *UriSource) {
		s.uri = uri
	}
}

func WithClipboardUri(enable bool) UriSourceOption {
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

func (s *UriSource) Uri() (string, error) {
	uris := 0

	if len(s.uri) > 0 {
		uris++
	}

	if s.cb {
		uris++
	}

	if uris > 1 {
		return "", errors.New("cannot read uri from multiple sources")
	}

	if len(s.uri) > 0 {
		return s.uri, nil
	}

	if s.cb {
		uri, err := ReadWcFromClipboard()
		if err != nil {
			return "", errors.Wrap(err, "failed to read uri from clipboard")
		}

		return uri, nil
	}

	return "", nil
}
