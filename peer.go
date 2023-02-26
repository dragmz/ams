package ams

type DynamicPeers struct {
}

type DynamicPeersOption func(p *DynamicPeers)

func MakeDynamicPeers(opts ...DynamicPeersOption) (*DynamicPeers, error) {
	p := &DynamicPeers{}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}
