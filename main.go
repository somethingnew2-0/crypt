package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/xordataexchange/crypt/backend/etcd"
	"github.com/xordataexchange/crypt/encoding/secconf"
)

var (
	data          string
	backend       string
	key           string
	keyring       string
	secretKeyring string
)

func init() {
	flag.StringVar(&backend, "backend", "", "backend")
	flag.StringVar(&data, "data", "", "path to configuration file")
	flag.StringVar(&key, "key", "", "configuration key")
	flag.StringVar(&keyring, "keyring", ".pubring.gpg", "path to public keyring")
	flag.StringVar(&secretKeyring, "secret-keyring", ".secring.gpg", "path to secret keyring")
}

func main() {
	flag.Parse()
	cmd := flag.Arg(0)
	backend := etcd.New([]string{"http://127.0.0.1:4001"})
	switch cmd {
	case "set":
		config, err := ioutil.ReadFile(data)
		if err != nil {
			log.Fatal(err)
		}
		kr, err := os.Open(keyring)
		if err != nil {
			log.Fatal(err)
		}
		defer kr.Close()
		secureValue, err := secconf.Encode(config, kr)
		if err != nil {
			log.Fatal(err)
		}
		if err := backend.Set(key, secureValue); err != nil {
			log.Fatal(err)
		}
	case "get":
		v, err := backend.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		skr, err := os.Open(secretKeyring)
		if err != nil {
			log.Fatal(err)
		}
		defer skr.Close()
		secureValue := []byte(v)
		value, err := secconf.Decode(secureValue, skr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\n", value)
	default:
		log.Fatal("unknown command: ", cmd)
	}
}