package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/sselph/scraper/ds"
	"github.com/sselph/scraper/ss"
)

func initGDB(system int32) bool {
	var sources []ds.DS
	hasher, err := ds.NewHasher(sha1.New, 1)
	if err != nil {
		fmt.Printf("hasher: %s", err)
		return false
	}
	hm, err := ds.CachedHashMap("", true)
	if err != nil {
		fmt.Println("hm: %s", err)
		return false
	}
	sources = append(sources, &ds.GDB{HM: hm, Hasher: hasher})
	sources = append(sources, &ds.ScummVM{HM: hm})
	sources = append(sources, &ds.Daphne{HM: hm})
	sources = append(sources, &ds.NeoGeo{HM: hm})
	scraper = &server{system: system, console: sources}
	return true
}

func initSS(system int32) bool {
	hasher, err := ds.NewHasher(sha1.New, 1)
	if err != nil {
		fmt.Printf("hasher: %s", err)
		return false
	}
	hm, err := ds.CachedHashMap("", true)
	if err != nil {
		fmt.Println("hm: %s", err)
		return false
	}
	dev, err := ss.DeobfuscateDevInfo()
	if err != nil {
		fmt.Printf("Can't access SS dev information: %q", err)
		return false
	}
	l := make(chan struct{}, 1)
	l <- struct{}{}
	ssDS := &ds.SS{
		HM:     hm,
		Hasher: hasher,
		Dev:    dev,
		Width:  400,
		Height: 400,
		Region: []string{"us", "wor", "eu", "jp", "fr", "xx"},
		Lang:   []string{"en"},
		Limit:  l,
	}
	ssMDS := &ds.SSMAME{
		Dev:    dev,
		Width:  400,
		Height: 400,
		Region: []string{"us", "wor", "eu", "jp", "fr", "xx"},
		Lang:   []string{"en"},
		Limit:  l,
	}
	scraper = &server{system: system, console: []ds.DS{ssDS}, arcade: []ds.DS{ssMDS}}
	return true
}

func initMAME(system int32) bool {
	mds, err := ds.NewMAME("", true)
	if err != nil {
		fmt.Println("mds: %s", err)
		return false
	}
	scraper = &server{system: system, arcade: []ds.DS{mds}}
	return true
}

//export SSelphInit
func SSelphInit(system int32) bool {
	if scraper != nil && system == scraper.system {
		return true
	}
	switch system {
	case 0:
		return initSS(system)
	case 1:
		return initGDB(system)
	case 2:
		return initMAME(system)
	}
	return false
}

//export SSelphClose
func SSelphClose() {
	if scraper != nil && scraper.system == 2 {
		scraper.arcade[0].(*ds.MAME).Close()
	}
	scraper = nil
	debug.FreeOSMemory()
}

//export SSelphReq
func SSelphReq(cPath, cPlatform, cImgPath *C.char) int32 {
	cLock.Lock()
	c := counter
	counter++
	cLock.Unlock()
	path := C.GoString(cPath)
	platform := C.GoString(cPlatform)
	imgPath := C.GoString(cImgPath)
	go func() {
		var arcade bool
		var output []byte
		defer func() {
			rLock.Lock()
			resp[c] = string(output)
			rLock.Unlock()
		}()
		if platform == "arcade" || platform == "neogeo" {
			arcade = true
		}
		g, err := scraper.scrape(path, imgPath, arcade)
		if err != nil {
			log.Print(err)
			return
		}
		output, err = xml.MarshalIndent(g, "  ", "    ")
		if err != nil {
			log.Print(err)
			return
		}
	}()
	return c
}

//export SSelphResp
func SSelphResp(c int32) (*C.char, bool) {
	rLock.Lock()
	r, ok := resp[c]
	defer rLock.Unlock()
	if !ok {
		return nil, false
	}
	delete(resp, c)
	return C.CString(r), true
}

func main() {}
