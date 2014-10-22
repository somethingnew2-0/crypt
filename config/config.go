package config

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/somethingnew2-0/crypt/backend"
	"github.com/somethingnew2-0/crypt/backend/consul"
	"github.com/somethingnew2-0/crypt/backend/etcd"
	"github.com/somethingnew2-0/crypt/encoding/secconf"
)

// A ConfigManager retrieves and decrypts configuration from a key/value store.
type ConfigManager interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	List(key string) ([]string, error)
	Watch(key string, stop chan bool) <-chan *Response
}

type configManager struct {
	keystore []byte
	store    backend.Store
}

// NewEtcdConfigManager returns a new ConfigManager backed by etcd.
func NewEtcdConfigManager(machines []string, keystore io.Reader) (ConfigManager, error) {
	store, err := etcd.New(machines)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(keystore)
	if err != nil {
		return nil, err
	}
	return configManager{bytes, store}, nil
}

// NewConsulConfigManager returns a new ConfigManager backed by consul.
func NewConsulConfigManager(machines []string, keystore io.Reader) (ConfigManager, error) {
	store, err := consul.New(machines)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(keystore)
	if err != nil {
		return nil, err
	}
	return configManager{bytes, store}, nil
}

// Get retrieves and decodes a secconf value stored at key.
func (c configManager) Get(key string) ([]byte, error) {
	value, err := c.store.Get(key)
	if err != nil {
		return nil, err
	}
	return secconf.Decode(value, bytes.NewBuffer(c.keystore))
}

// Set endcodes and stores a secconf value at key.
func (c configManager) Set(key string, value []byte) error {
	encodedValue, err := secconf.Encode(value, bytes.NewBuffer(c.keystore))
	if err != nil {
		return err
	}
	return c.store.Set(key, encodedValue)
}

func (c configManager) List(key string) ([]string, error) {
	return c.store.List(key)
}

type Response struct {
	Value []byte
	Error error
}

func (c configManager) Watch(key string, stop chan bool) <-chan *Response {
	resp := make(chan *Response, 0)
	backendResp := c.store.Watch(key, stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case r := <-backendResp:
				if r.Error != nil {
					resp <- &Response{nil, r.Error}
					continue
				}
				value, err := secconf.Decode(r.Value, bytes.NewBuffer(c.keystore))
				resp <- &Response{value, err}
			}
		}
	}()
	return resp
}
