package etcd

import (
	"errors"
	"fmt"
	"time"

	"github.com/xordataexchange/crypt/backend"

	goetcd "github.com/coreos/go-etcd/etcd"
)

type Client struct {
	client    *goetcd.Client
	waitIndex uint64
}

func New(machines []string) (*Client, error) {
	return &Client{goetcd.NewClient(machines), 0}, nil
}

func (c *Client) Get(key string) ([]byte, error) {
	resp, err := c.client.Get(key, false, false)
	if err != nil {
		return nil, err
	}
	return []byte(resp.Node.Value), nil
}

func (c *Client) Set(key string, value []byte) error {
	_, err := c.client.Set(key, string(value), 0)
	return err
}

func (c *Client) List(key string) ([]string, error) {
	resp, err := c.client.Get(key, false, false)
	if err != nil {
		return nil, err
	}
	if !resp.Node.Dir {
		return nil, errors.New(fmt.Sprintf("Key \"%s\" is not a directory", key))
	}

	entries := []string{}
	for _, node := range resp.Node.Nodes {
		if !node.Dir {
			entries = append(entries, node.Key)
		}
	}

	return entries, nil
}

func (c *Client) Watch(key string, stop chan bool) <-chan *backend.Response {
	respChan := make(chan *backend.Response, 0)
	go func() {
		for {
			var resp *goetcd.Response
			var err error
			if c.waitIndex == 0 {
				resp, err = c.client.Get(key, false, false)
				if err != nil {
					respChan <- &backend.Response{nil, err}
					time.Sleep(time.Second * 5)
					continue
				}
				c.waitIndex = resp.EtcdIndex
				respChan <- &backend.Response{[]byte(resp.Node.Value), nil}
			}
			resp, err = c.client.Watch(key, c.waitIndex+1, false, nil, stop)
			if err != nil {
				respChan <- &backend.Response{nil, err}
				time.Sleep(time.Second * 5)
				continue
			}
			c.waitIndex = resp.Node.ModifiedIndex
			respChan <- &backend.Response{[]byte(resp.Node.Value), nil}
		}
	}()
	return respChan
}
