package main

import (
	"log"
)

type Module struct {
	Id       string
	Name     string
	InitFunc func() error
}

var modules []Module

func init() {
	modules = []Module{
		{
			Id:       "herenow",
			Name:     "HereNow",
			InitFunc: nil,
		},
	}
}

func StartModules() {
	for _, m := range modules {

		log.Printf("\t%s...", m.Name)

		if m.InitFunc == nil {
			continue;
		}

		if err := m.InitFunc(); err != nil {
			panic(err)
		}
	}
}
