package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

var bck *Backend

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "3.2.9",
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		var err error

		bck, err = NewBackend(fmt.Sprintf(":%s", resource.GetPort("6379/tcp")))
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	bck.Close()
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

type errorHandler interface {
	Error(args ...interface{})
}

func getBackend(t errorHandler) (*Backend, func()) {
	return bck, func() {
		conn := bck.pool.Get()
		defer conn.Close()
		_, err := conn.Do("FLUSHDB")
		if err != nil {
			t.Error(err)
		}
	}
}

func TestBackend(t *testing.T) {
	backend, cleanup := getBackend(t)
	defer cleanup()

	topic, err := backend.Topic("a")
	require.NoError(t, err)
	require.NotNil(t, topic)
	require.NotNil(t, topic.(*Topic).conn)

	err = topic.Close()
	require.NoError(t, err)

	b1, err := backend.Topic("a")
	require.NoError(t, err)

	b2, err := backend.Topic("b")
	require.NoError(t, err)

	err = b1.Close()
	require.NoError(t, err)

	err = b2.Close()
	require.NoError(t, err)
}