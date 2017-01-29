package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/sselph/scraper/ds"
	"github.com/sselph/scraper/rom"
)

var scraper *server
var resp map[int32]string
var counter int32
var cLock sync.Mutex
var rLock sync.Mutex

func init() {
	resp = make(map[int32]string)
}

type game struct {
	GameTitle   string
	Overview    string
	ImageURL    string
	ImagePath   string
	Image       ds.Image `xml:"-"`
	Rating      float64
	ReleaseDate string
	Developer   string
	Publisher   string
	Genre       string
	Players     string
	CRCMatch    bool
}

func toGame(g *ds.Game, arcade bool) game {
	out := game{
		GameTitle:   g.GameTitle,
		Overview:    g.Overview,
		Rating:      g.Rating,
		ReleaseDate: g.ReleaseDate,
		Developer:   g.Developer,
		Publisher:   g.Publisher,
		Genre:       g.Genre,
	}
	if arcade {
		var img ds.Image
		var ok bool
		for _, it := range []ds.ImgType{"t", "m", "s"} {
			if img, ok = g.Thumbs[it]; ok {
				break
			}
		}
		out.Image = img
	} else {
		out.Image = g.Thumbs["b"]
	}
	if g.Players > 0 {
		out.Players = strconv.FormatInt(g.Players, 10)
	}
	return out
}

type server struct {
	system  int32
	console []ds.DS
	arcade  []ds.DS
}

func (s *server) scrape(path, imgPath string, arcade bool) ([]game, error) {
	if s == nil {
		return nil, nil
	}
	r, err := rom.NewROM(path)
	if err != nil {
		return nil, err
	}
	for try := 0; try <= 3; try++ {
		if arcade && r.Ext != ".zip" && r.Ext != ".7z" {
			return nil, fmt.Errorf("extension unknown: %s", r.Ext)
		}
		if arcade {
			err = r.GetGame(s.arcade, &rom.GameOpts{AddNotFound: true})
		} else {
			err = r.GetGame(s.console, &rom.GameOpts{AddNotFound: true})
		}
		if err != nil {
			log.Print(err)
			continue
		}
		if r.Game == nil {
			return nil, nil
		}
		g := toGame(r.Game, arcade)
		g.CRCMatch = true
		if imgPath != "" && g.Image != nil {
			if err := g.Image.Save(imgPath, 400, 400); err == nil {
				g.ImagePath = imgPath
			}
		}
		if g.ImagePath == "" {
			switch i := g.Image.(type) {
			case ds.HTTPImage:
				g.ImageURL = i.URL
			case ds.HTTPImageSS:
				g.ImageURL = i.URL
			}
		}
		return []game{g}, err
	}
	return nil, err
}
