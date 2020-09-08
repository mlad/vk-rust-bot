package main

/*

TODO:
-- Мьютексы для апдейтера серверов

*/

import (
	"encoding/json"
	"github.com/tidwall/gjson"
	"log"
	"math/rand"
	"strings"
	"time"
)

var VkCommands map[string]func(vk *Vk, object *LongPollMessage)

func onLongPollMessage(vk *Vk, object gjson.Result) {
	// todo: исправить кашу с типами данных
	msg := new(LongPollMessage)
	_ = json.Unmarshal([]byte(object.Raw), msg)

	Command := gjson.Get(msg.Payload, "command")
	if Command.Exists() {
		if event, ok := VkCommands[Command.String()]; ok {
			event(vk, msg)
		}

		return
	}

	if strings.EqualFold(msg.Text, "старт") || strings.EqualFold(msg.Text, "привет") || strings.EqualFold(msg.Text, "начать") {
		CommandStart(vk, msg)
	} else {
		vk.SendMessage(msg.FromId, "Привет! 👋\nЭто бот сообщества LIVE RUST\n\n❗ Если у тебя нет меню, отправь текст \"старт\"")
	}
}

func CommandStart(vk *Vk, object *LongPollMessage) {
	kb := VkKeyboard{OneTime: false}

	kb.AddRow(kb.TxtBtn("🔍 Подобрать Rust сервер", "secondary", `{"command":"rustFind"}`))
	kb.AddRow(kb.TxtBtn("📣 Реклама в LIVE RUST", "secondary", `{"command":"ads"}`))

	vk.SendKeyboard(object.FromId, "Выбери действие в меню", &kb)
}

func CommandAds(vk *Vk, object *LongPollMessage) {
	vk.SendMessage(object.FromId, "По рекламным вопросам обращайтесь в [liveadv|LIVE AD]")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg, err := readConfig(configFilePath)
	if err != nil {
		log.Fatalf("Read config error: %v\n", err)
	}

	LoadRustServers()

	VkCommands = map[string]func(vk *Vk, object *LongPollMessage){
		"start":    CommandStart,
		"rustFind": CommandRustFind,
		"ads":      CommandAds,
	}

	vk := Vk{accessToken: cfg.Token, version: cfg.Version}
	vk.GroupPoll(cfg.GroupId, onLongPollMessage)

	RunApiServer()

	ch := make(chan int)
	<-ch
}
