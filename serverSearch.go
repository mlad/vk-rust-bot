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
	{Message: "Какого жанра сервер ты хочешь?", Variants: []string{"Моды", "Классика", "Развлекательные"},
		Checker: func(answer int, s *RustServerInfo) bool {
			return answer == s.Rate
		}},
	{Message: "Какие рейты ты хочешь?", Variants: []string{"x1 (стандарт  классики)", "x2 - x3 (стандарт модов)", "x4 - x9 (быстрая игра)", "x10 и выше"},
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
	{Message: "Сколько игроков в твоей команде?", Variants: []string{"1", "2 - 3", "4 - 5", "6 и более (клан)"},
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
	{Message: "Какая карта должна быть на сервере?", Variants: []string{"Любая", "Procedural", "Barren", "Hapis Island"},
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
	{Message: "Насколько часто должен быть вайп?", Variants: []string{"Без разницы", "Каждые 3-5 дней", "Каждые 7 дней", "Каждые 14 дней"},
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
	if server.MaxPlayers == 0 { // Сервер оффлайн
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

	// Проверка номера этапа

	step := payload.Step

	if step < 0 || step > len(ServerSearchAnswers) {
		return
	}

	// Если был выбран ответ (т.е не первый вывод меню)

	if payload.Answer != 0 && step != len(ServerSearchAnswers) {
		if payload.Answer < 1 || payload.Answer > len(ServerSearchAnswers[step].Variants) {
			return
		}

		payload.Data = append(payload.Data, payload.Answer)

		payload.Step++
		step++

		if step == 1 {
			if payload.Answer == 2 { // Классика, пропускаем этап выбора рейтов
				payload.Data = append(payload.Data, 1)
				payload.Step++
				step++
			} else if payload.Answer == 3 { // Развлекательные, пропускаем остальные этапы
				payload.Data = append(payload.Data, 0, 0, 0, 0)
				payload.Step = len(ServerSearchAnswers)
				step = payload.Step
			}
		}
	}

	// Все вопросы обработаны. Ищем сервер

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
			// Перемешиваем найденные сервера

			rand.Shuffle(len(good), func(i, j int) { good[i], good[j] = good[j], good[i] })

			// Стараемся вывести сервера от разных проектов

			unique := make(map[uint32]bool)
			count := len(good)

			gFinal := [3]*RustServerInfo{} // Финальный список. Три сервера, которые будут выведены
			fPtr := 0                      // Счетчик для финального списка

			for i := 0; i < count; i++ {
				s := good[i]
				if _, ok := unique[s.Key]; !ok {
					unique[s.Key] = true // Отмечаем, что сервер от проекта добавлен

					// "Удаляем" элемент из среза
					good[i], good[count-1] = good[count-1], good[i]
					count--

					// Добавляем элемент в финальный список
					gFinal[fPtr] = s
					fPtr++
					if fPtr == 3 {
						break
					}
				}
			}

			for ; fPtr < 3; fPtr++ { // todo: заменить на " fPtr++ < 3 " ?
				if count == 0 {
					break
				}
				j := rand.Intn(count)

				gFinal[fPtr] = good[j]

				// "Удаляем" элемент из среза. Чтобы в очередной раз не выбрать тот же
				good[j], good[count-1] = good[count-1], good[j]
				count--
			}

			// Вывод списка найденных сервероа

			resp.WriteString("🔍 По выбранным параметрам найдены сервера:\n\n")

			for i := 0; i < fPtr; i++ {
				j := gFinal[i]
				resp.WriteString(fmt.Sprintf("▶ connect %s\n%s\nОнлайн %d / %d -- Вайп %s\n\n",
					j.Address, j.Hostname, j.Players, j.MaxPlayers, time.Unix(j.Wiped, 0).Format("02.01 в 15:04")))
			}
		} else {
			resp.WriteString("😢 Ничего не найдено. Попробуй другие параметры")
		}

		if len(good) > 3 {
			pl, _ := json.Marshal(payload)
			kb.AddRow(kb.TxtBtn("🔄 Покажите другие сервера", "secondary", string(pl)))
		}
		kb.AddRow(kb.TxtBtn("🔍 Попробовать снова", "secondary", `{"command":"rustFind"}`))
		kb.AddRow(kb.TxtBtn("🏠 Вернуться на главную", "positive", `{"command":"start"}`))
		vk.SendKeyboard(object.FromId, resp.String(), &kb)
		return
	}

	// Выводим очередные варианты ответов на вопрос

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

	kb.AddRow(kb.TxtBtn("🏠 Вернуться на главную", "positive", `{"command":"start"}`))
	vk.SendKeyboard(object.FromId, ServerSearchAnswers[step].Message, &kb)
}
