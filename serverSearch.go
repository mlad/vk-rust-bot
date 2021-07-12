package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type ServerSearchPayload struct {
	Command string `json:"command"`
	Answer  int    `json:"num"`
	Step    int    `json:"step"`
	Data    []int  `json:"data"`
}

type ServerSearchAnswer struct {
	Message  string
	Variants []string
	Checker  func(answer int, s *RustServerInfo) bool
}

var ServerSearchAnswers = [...]ServerSearchAnswer{
	{Message: "Server genre?", Variants: []string{"Modded", "Classic", "Fun"},
		Checker: func(answer int, s *RustServerInfo) bool {
			return answer == s.Rate
		}},
	{Message: "Server rates?", Variants: []string{"x1 (classic)", "x2 - x3 (mods)", "x4 - x9 (fast play)", "x10 and higher"},
		Checker: func(answer int, s *RustServerInfo) bool {
			switch answer {
			case 1:
				return s.Rate == 1
			case 2:
				return s.Rate >= 2 && s.Rate <= 3
			case 3:
				return s.Rate >= 4 && s.Rate <= 9
			case 4:
				return s.Rate >= 10
			default:
				return false
			}
		}},
	{Message: "How many players are on your team?", Variants: []string{"1", "2 - 3", "4 - 5", "6 and more (clan)"},
		Checker: func(answer int, s *RustServerInfo) bool {
			if s.Genre == GenreClassic && s.MaxTeam == 100 { // На классике без ограничения лимита не проверяем
				return true
			}

			switch answer {
			case 1:
				return s.MaxTeam == 1
			case 2:
				return s.MaxTeam >= 2 && s.MaxTeam <= 3
			case 3:
				return s.MaxTeam >= 4 && s.MaxTeam <= 5
			case 4:
				return s.MaxTeam >= 6
			default:
				return false
			}
		}},
	{Message: "Server map?", Variants: []string{"Any", "Procedural", "Barren", "Hapis Island"},
		Checker: func(answer int, s *RustServerInfo) bool {
			switch answer {
			case 1:
				return true
			case 2:
				return s.Map[0] == 'P'
			case 3:
				return s.Map[0] == 'B'
			case 4:
				return s.Map[0] == 'H'
			default:
				return false
			}
		}},
	{Message: "Wipe interval?", Variants: []string{"Any", "Every 3-5 days", "Every 7 days", "Every 14 days"},
		Checker: func(answer int, s *RustServerInfo) bool {
			switch answer {
			case 1:
				return true
			case 2:
				return s.WipeInterval <= 5
			case 3:
				return s.WipeInterval == 7
			case 4:
				return s.WipeInterval == 14
			default:
				return false
			}
		}},
}

func (server *RustServerInfo) CheckParams(genre byte, params ...int) bool {
	if server.MaxPlayers == 0 { // Server is offline
		return false
	}

	if genre != server.Genre {
		return false
	}

	if genre == GenreFun {
		return true
	}

	for k, v := range params {
		if !ServerSearchAnswers[k+1].Checker(v, server) {
			return false
		}
	}

	return true
}

func CommandRustFind(vk *Vk, object *LongPollMessage) {
	kb := VkKeyboard{OneTime: false}

	payload := new(ServerSearchPayload)
	if err := json.Unmarshal([]byte(object.Payload), payload); err != nil {
		return
	}

	// Check step id

	step := payload.Step

	if step < 0 || step > len(ServerSearchAnswers) {
		return
	}

	// An answer was chosen (i.e not first menu page)

	if payload.Answer != 0 && step != len(ServerSearchAnswers) {
		if payload.Answer < 1 || payload.Answer > len(ServerSearchAnswers[step].Variants) {
			return
		}

		payload.Data = append(payload.Data, payload.Answer)

		payload.Step++
		step++

		if step == 1 {
			if payload.Answer == 2 { // Classics, skip rate selection page
				payload.Data = append(payload.Data, 1)
				payload.Step++
				step++
			} else if payload.Answer == 3 { // Fun, skip all next pages
				payload.Data = append(payload.Data, 0, 0, 0, 0)
				payload.Step = len(ServerSearchAnswers)
				step = payload.Step
			}
		}
	}

	// All questions have been processed. Looking for a server

	if step == len(ServerSearchAnswers) {

		d := payload.Data
		if len(d) != 5 {
			return
		}

		good := make([]*RustServerInfo, 0, 30)

		for i, j := 0, len(RustServers); i < j; i++ {
			v := &RustServers[i]

			if v.CheckParams(byte(d[0]), d[1:]...) {
				good = append(good, v)
			}
		}

		resp := strings.Builder{}
		if len(good) != 0 {
			// Shuffle founded servers

			rand.Shuffle(len(good), func(i, j int) { good[i], good[j] = good[j], good[i] })

			// Trying to show servers from different projects

			unique := make(map[uint32]bool)
			count := len(good)

			gFinal := [3]*RustServerInfo{} // Final list. Three servers that will be shown
			fPtr := 0                      // Counter for the final list

			for i := 0; i < count; i++ {
				s := good[i]
				if _, ok := unique[s.Key]; !ok {
					unique[s.Key] = true

					// Delete element from slice
					good[i], good[count-1] = good[count-1], good[i]
					count--

					// Add element to final array
					gFinal[fPtr] = s
					fPtr++
					if fPtr == 3 {
						break
					}
				}
			}

			for ; fPtr < 3; fPtr++ {
				if count == 0 {
					break
				}
				j := rand.Intn(count)

				gFinal[fPtr] = good[j]

				// Remove element from slice
				good[j], good[count-1] = good[count-1], good[j]
				count--
			}

			// Show found servers

			resp.WriteString("🔍 Found servers::\n\n")

			for i := 0; i < fPtr; i++ {
				j := gFinal[i]
				resp.WriteString(fmt.Sprintf("▶ connect %s\n%s\nPlayers %d / %d -- Wiped %s\n\n",
					j.Address, j.Hostname, j.Players, j.MaxPlayers, time.Unix(j.Wiped, 0).Format("02.01 at 15:04")))
			}
		} else {
			resp.WriteString("😢 Nothing found. Try other parameters")
		}

		if len(good) > 3 {
			pl, _ := json.Marshal(payload)
			kb.AddRow(kb.TxtBtn("🔄 Show other servers", "secondary", string(pl)))
		}
		kb.AddRow(kb.TxtBtn("🔍 Try again", "secondary", `{"command":"rustFind"}`))
		kb.AddRow(kb.TxtBtn("🏠 Main page", "positive", `{"command":"start"}`))
		vk.SendKeyboard(object.FromId, resp.String(), &kb)
		return
	}

	// Display next parameters for answers to the question

	count := len(ServerSearchAnswers[step].Variants)
	buttons := make([]VkButton, 0, count)
	buttonExists := make([]bool, count)
	totalExists := 0

	if step != 0 {
		tmp := append(payload.Data[1:], 0)
	buttonExistsFor:
		for i, j := 0, len(RustServers); i < j; i++ {

			/*if !RustServers[i].CheckParams(byte(payload.Data[0]), payload.Data[1:]...) {
				continue
			}*/

			// todo: вероятно, достаточно будет общего чека на сервер и отдельного чека только k + 1 параметра (будущего)

			// tmp := append(payload.Data[1:], 0)
			for k := 0; k < count; k++ {
				tmp[len(tmp)-1] = k + 1
				if !buttonExists[k] && RustServers[i].CheckParams(byte(payload.Data[0]), tmp...) {
					buttonExists[k] = true
					totalExists++

					if totalExists == count {
						break buttonExistsFor
					}
				}
			}
		}
	} else {
		for i := 0; i < count; i++ {
			buttonExists[i] = true
		}
	}

	for i := 0; i < count; i++ {
		if buttonExists[i] {
			payload.Answer = i + 1
			pl, _ := json.Marshal(payload)

			buttons = append(buttons, kb.TxtBtn(ServerSearchAnswers[step].Variants[i], "secondary", string(pl)))
		}
	}

	count = len(buttons)
	switch {
	case count%2 == 0:
		for i := 0; i < count; i += 2 {
			kb.AddRow(buttons[i], buttons[i+1])
		}
	default:
		for i := 0; i < count; i++ {
			kb.AddRow(buttons[i])
		}
	}

	kb.AddRow(kb.TxtBtn("🏠 Main page", "positive", `{"command":"start"}`))
	vk.SendKeyboard(object.FromId, ServerSearchAnswers[step].Message, &kb)
}
