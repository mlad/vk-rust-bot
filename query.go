package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"hash/crc32"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var RustServers []RustServerInfo

type RustServerInfo struct {
	Address      string
	Hostname     string
	Map          string
	MaxPlayers   int
	Players      int
	Wiped        int64
	MaxTeam      int
	Rate         int
	WipeInterval int
	Genre        byte
	Key          uint32
}

const (
	GenreMod     byte = 1
	GenreClassic byte = 2
	GenreFun     byte = 3
)

func (server *RustServerInfo) UpdateInfo() {
	conn, err := net.Dial("udp", server.Address)
	if err != nil {
		fmt.Printf("Update server %s dial error: %s\n", server.Address, err.Error())
		return
	}

	_, err = conn.Write([]byte("\xFF\xFF\xFF\xFFTSource Engine Query\x00"))
	if err != nil {
		fmt.Printf("Update server %s write error: %s\n", server.Address, err.Error())
		return
	}

	r := bufio.NewReader(conn)

	_, _ = r.Discard(4 + 1 + 1)                     // header + protocol
	server.Hostname, _ = r.ReadString(0)            // name
	server.Map, _ = r.ReadString(0)                 // map
	_, _ = r.ReadString(0)                          // folder
	_, _ = r.ReadString(0)                          // game
	_, _ = r.Discard(2 + 1 + 1 + 1 + 1 + 1 + 1 + 1) // app id + Players + MaxPlayers + Bots + Dedicated + Os + Password + Secure
	_, _ = r.ReadString(0)

	edf, _ := r.ReadByte()
	if edf&0x80 != 0 {
		_, _ = r.Discard(2) // Game port
	}

	if edf&0x10 != 0 {
		_, _ = r.Discard(8) // SteamID
	}

	if edf&0x40 != 0 {
		_, _ = r.Discard(2)    // SpecPort
		_, _ = r.ReadString(0) // SpecName
	}

	if edf&0x20 != 0 {
		keywords, _ := r.ReadString(0) // Keywords

		for _, i := range strings.Split(keywords, ",") {
			switch {
			case strings.HasPrefix(i, "mp"):
				server.MaxPlayers, _ = strconv.Atoi(i[2:])
			case strings.HasPrefix(i, "cp"):
				server.Players, _ = strconv.Atoi(i[2:])
			case strings.HasPrefix(i, "born"):
				server.Wiped, _ = strconv.ParseInt(i[4:], 10, 64)
			}
		}
	}

	if edf&0x01 != 0 {
		_, _ = r.Discard(8) // GameID
	}

	_ = conn.Close()
}

func LoadRustServers() {
	fp, err := os.Open(ServersFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Server list file not found!\n"+
				"Create file \"%s\" and fill it with the following structure:\n"+
				"IP:PORT<TAB>TEAM_PLAYERS<TAB>RATES<TAB>WIPE_INTERVAL<TAB>GENRE<TAB>PROJECT_NAME<NEW LINE>\n"+
				"(Each server on a new line)\n"+
				"Genres: m=modded, c=classic, f=fun\n", ServersFilePath)
		}
		log.Fatalf("Server list open error: %s\n", err.Error())
	}

	reader := csv.NewReader(fp) // address,max_players,rate,wipe_interval
	reader.Comment = '#'
	reader.Comma = '\t'

	data, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Server list struct error: %s\n", err.Error())
	}

	RustServers = make([]RustServerInfo, len(data))

	for k, v := range data {
		srv := &RustServers[k]
		srv.Address = v[0]
		srv.MaxTeam, _ = strconv.Atoi(v[1])
		srv.Rate, _ = strconv.Atoi(v[2])
		srv.WipeInterval, _ = strconv.Atoi(v[3])

		switch v[4][0] {
		case 'm':
			srv.Genre = GenreMod
		case 'c':
			srv.Genre = GenreClassic
		case 'f':
			srv.Genre = GenreFun
		}

		srv.Key = crc32.ChecksumIEEE([]byte(v[5]))
	}

	RunRustServerUpdate()
}

func RunRustServerUpdate() {
	go func() {
		filter1 := regexp.MustCompile("([^ ])\\|([^ ])")                // Filter links to VK pages
		filter2 := regexp.MustCompile("([\\w-]+)\\.(ru|RU|su|com|net)") // Filter site links

		ch := make(chan bool, 10)
		tm := time.NewTicker(30 * time.Second)

		updSrv := func(srv *RustServerInfo, locker <-chan bool) {
			srv.UpdateInfo()

			if srv.MaxPlayers == 0 {
				log.Printf("Server %s return empty data. Check it\n", srv.Address)
			}

			srv.Hostname = filter1.ReplaceAllString(srv.Hostname, "$1 | $2")
			srv.Hostname = filter2.ReplaceAllString(srv.Hostname, "$1·$2")

			<-locker
		}

		for {
			for i, j := 0, len(RustServers); i < j; i++ {
				ch <- true
				go updSrv(&RustServers[i], ch)
			}

			<-tm.C
		}
	}()
}
