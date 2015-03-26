package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/davidmz/logg"
	"github.com/vaughan0/go-ini"
)

type Conf struct {
	ListenAddr  string
	ListenMemc  string
	LogLevel    logg.Level
	ChannelSize int
	MsgLifetime time.Duration
	Secret      []byte
	Sites       []*SiteConf
	NSessions   uint32
}

type SiteConf struct {
	Name       string
	Secret     []byte
	PostSecret []byte
}

func ReadConf() (*Conf, error) {
	var (
		showHelp     bool
		confFileName string
		ok           bool
	)

	flag.StringVar(&confFileName, "c", "", "config file name")
	flag.BoolVar(&showHelp, "h", false, "show this help")
	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}
	if confFileName == "" {
		fmt.Fprintln(os.Stderr, "Error:", "config file name required")
		flag.Usage()
		os.Exit(1)
	}

	confFile, err := ini.LoadFile(confFileName)
	if err != nil {
		return nil, err
	}

	conf := &Conf{}

	baseSection := confFile[""]
	delete(confFile, "")

	if conf.ListenAddr, ok = baseSection["listen"]; !ok {
		return nil, fmt.Errorf("listen address not setted")
	}

	conf.ListenMemc = baseSection["listen_memcache"]

	if x, err := logg.LevelByName(baseSection["log_level"]); err != nil {
		return nil, err
	} else {
		conf.LogLevel = x
	}

	if x, err := strconv.ParseUint(baseSection["channel_size"], 10, 64); err != nil {
		return nil, err
	} else {
		conf.ChannelSize = int(x)
	}

	if x, err := time.ParseDuration(baseSection["message_lifetime"]); err != nil {
		return nil, err
	} else if x < 0 {
		return nil, fmt.Errorf("negative message_lifetime")
	} else {
		conf.MsgLifetime = x
	}

	if x, err := base64.StdEncoding.DecodeString(baseSection["secret"]); err != nil {
		return nil, err
	} else {
		conf.Secret = x
	}

	// Sites
	for name, sect := range confFile {
		s := &SiteConf{Name: name}

		if x, err := base64.StdEncoding.DecodeString(sect["secret"]); err != nil {
			return nil, err
		} else {
			s.Secret = x
		}

		if x, err := base64.StdEncoding.DecodeString(sect["post_secret"]); err != nil {
			return nil, err
		} else {
			s.PostSecret = x
		}

		conf.Sites = append(conf.Sites, s)
	}

	return conf, nil
}
