package rpc

import (
	"context"
	"io"

	"github.com/asdine/lobby"
	"github.com/asdine/lobby/rpc/proto"
	"google.golang.org/grpc"
)

var _ lobby.Backend = new(Backend)

// NewBackend returns a gRPC backend. It is used to communicate with external backends.
func NewBackend(addr string) (*Backend, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := proto.NewBucketServiceClient(conn)

	return &Backend{
		conn:   conn,
		client: client,
	}, nil
}

// Backend is a gRPC backend.
type Backend struct {
	conn   *grpc.ClientConn
	client proto.BucketServiceClient
}

// Bucket returns the bucket associated with the given id.
func (s *Backend) Bucket(name string) (lobby.Bucket, error) {
	return NewBucket(name, s.client), nil
}

// Close the connexion to the backend.
func (s *Backend) Close() error {
	return s.conn.Close()
}

var _ lobby.Bucket = new(Bucket)

// NewBucket returns a Bucket.
func NewBucket(name string, client proto.BucketServiceClient) *Bucket {
	return &Bucket{
		name:   name,
		client: client,
	}
}

// Bucket is a gRPC implementation of a bucket.
type Bucket struct {
	name   string
	client proto.BucketServiceClient
}

// Put value to the bucket. Returns an Item.
func (b *Bucket) Put(key string, value []byte) (*lobby.Item, error) {
	stream, err := b.client.Put(context.Background())
	if err != nil {
		return nil, err
	}

	err = stream.Send(&proto.NewItem{
		Bucket: b.name,
		Item: &proto.Item{
			Key:   key,
			Value: value,
		},
	})
	if err != nil {
		return nil, errFromGRPC(err)
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return nil, errFromGRPC(err)
	}

	return &lobby.Item{
		Key:   key,
		Value: value,
	}, nil
}

// Get an item by key.
func (b *Bucket) Get(key string) (*lobby.Item, error) {
	item, err := b.client.Get(context.Background(), &proto.Key{Bucket: b.name, Key: key})
	if err != nil {
		return nil, errFromGRPC(err)
	}

	return &lobby.Item{
		Key:   item.Key,
		Value: item.Value,
	}, nil
}

// Delete item from the bucket
func (b *Bucket) Delete(key string) error {
	_, err := b.client.Delete(context.Background(), &proto.Key{Bucket: b.name, Key: key})
	if err != nil {
		return errFromGRPC(err)
	}

	return nil
}

// Page returns a list of items
func (b *Bucket) Page(page int, perPage int) ([]lobby.Item, error) {
	stream, err := b.client.List(context.Background(), &proto.Page{
		Bucket:  b.name,
		Page:    int32(page),
		PerPage: int32(perPage),
	})
	if err != nil {
		return nil, errFromGRPC(err)
	}

	var items []lobby.Item
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errFromGRPC(err)
		}
		items = append(items, lobby.Item{
			Key:   item.Key,
			Value: item.Value,
		})
	}

	return items, nil
}

// Close the bucket session.
func (b *Bucket) Close() error {
	return nil
}