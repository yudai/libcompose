package client

import (
	"sync"

	"github.com/docker/libcompose/project"
)

// Factory is a factory to create docker clients.
type Factory interface {
	// Create constructs a Docker client for the given service. The passed in
	// config may be nil in which case a generic client for the project should
	// be returned.
	// Closing the client is caller's responsibility.
	Create(service project.Service) (APIClientCloser, error)
}

type defaultFactory struct {
	opts Options

	client APIClientCloser
	count  int64
	mutex  *sync.Mutex
}

// NewDefaultFactory creates and returns the default client factory that uses
// github.com/docker/docker client.
func NewDefaultFactory(opts Options) (Factory, error) {
	return &defaultFactory{
		opts:  opts,
		count: 0,
		mutex: new(sync.Mutex),
	}, nil
}

func (s *defaultFactory) Create(service project.Service) (APIClientCloser, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.count == 0 {
		client, err := Create(s.opts)
		if err != nil {
			return nil, err
		}
		s.client = client
	}

	s.count++

	return &CachedClient{
		APIClientCloser: s.client,
		factory:         s,
	}, nil
}

type CachedClient struct {
	APIClientCloser
	factory *defaultFactory
}

func (client *CachedClient) Close() error {
	client.factory.mutex.Lock()
	defer client.factory.mutex.Unlock()

	client.factory.count--

	if client.factory.count == 0 {
		return client.APIClientCloser.Close()
	}
	return nil
}
