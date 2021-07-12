package main

import (
	"encoding/json"
	"github.com/tidwall/gjson"
	"log"
	"math/rand"
	"strings"
	"time"
)

const ConfigFilePath string = "config.json"
const ServersFilePath string = "server.tsv"
const VkApiVersion = "5.85"

var Config *ConfigData
var VkCommands map[string]func(vk *Vk, object *LongPollMessage)

func onLongPollMessage(vk *Vk, object gjson.Result) {
	msg := new(LongPollMessage)
	_ = json.Unmarshal([]byte(object.Raw), msg)

	Command := gjson.Get(msg.Payload, "command")
	if Command.Exists() {
		if event, ok := VkCommands[Command.String()]; ok {
			event(vk, msg)
		}

		return
	}

	for _, i := range Config.StartCommands {
		if strings.EqualFold(msg.Text, i) {
			CommandStart(vk, msg)
			return
		}
	}

	vk.SendMessage(msg.FromId, Config.WelcomeMessage)
}

func CommandStart(vk *Vk, object *LongPollMessage) {
	kb := VkKeyboard{OneTime: false}

	kb.AddRow(kb.TxtBtn("🔍 Find Rust servers", "secondary", `{"command":"rustFind"}`))
	kb.AddRow(kb.TxtBtn("📣 Example command", "secondary", `{"command":"example"}`))

	vk.SendKeyboard(object.FromId, "Select a menu item:", &kb)
}

func CommandExample(vk *Vk, object *LongPollMessage) {
	vk.SendMessage(object.FromId, "Example command response")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	var err error
	if Config, err = ConfigRead(ConfigFilePath); err != nil {
		log.Fatalf("Config read error: %s\n", err.Error())
	}

	LoadRustServers()

	VkCommands = map[string]func(vk *Vk, object *LongPollMessage){
		"start":    CommandStart,
		"rustFind": CommandRustFind,
		"example":  CommandExample,
	}

	vk := Vk{accessToken: Config.Token, version: VkApiVersion}
	vk.GroupPoll(Config.GroupId, onLongPollMessage)

	ch := make(chan int)
	<-ch
}
