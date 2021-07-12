package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Vk struct {
	accessToken string
	version     string
}

type vkLongPollServer struct {
	Key    string
	Server string
	Ts     string
}

func (v *Vk) Request(method string, params url.Values) (response gjson.Result, err error) {
	params.Add("access_token", v.accessToken)
	params.Add("v", v.version)

	resp, err := http.PostForm(fmt.Sprintf("https://api.vk.com/method/%s", method), params)
	if err != nil {
		return gjson.Result{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return gjson.Result{}, fmt.Errorf("bad http code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return gjson.Result{}, err
	}

	result := gjson.GetBytes(body, "response")
	if !result.Exists() {
		errorMsg := gjson.GetBytes(body, "error.error_msg")
		if !errorMsg.Exists() {
			return gjson.Result{}, errors.New("unknown json struct error")
		}

		return gjson.Result{}, errors.New(errorMsg.Str)
	}

	return result, nil
}

// https://vk.com/dev/messages.send
func (v *Vk) SendMessage(peerId int64, text string) {
	_, err := v.Request("messages.send", url.Values{
		"peer_id":   {strconv.FormatInt(peerId, 10)},
		"random_id": {strconv.FormatInt(rand.Int63(), 10)},
		"message":   {text},
	})

	if err != nil {
		log.Println("SendMessage error:", err)
	}
}

// https://vk.com/dev/messages.send
// https://vk.com/dev/bots_docs_3
func (v *Vk) SendKeyboard(peerId int64, text string, keyboard *VkKeyboard) {
	kb, _ := json.Marshal(keyboard)

	_, err := v.Request("messages.send", url.Values{
		"peer_id":          {strconv.FormatInt(peerId, 10)},
		"random_id":        {strconv.FormatInt(rand.Int63(), 10)},
		"message":          {text},
		"keyboard":         {string(kb)},
		"dont_parse_links": {"1"},
		"disable_mentions": {"1"},
	})

	if err != nil {
		log.Println("SendKeyboard error:", err)
	}
}

func (v *Vk) GroupPoll(groupId string, event func(vk *Vk, message gjson.Result)) {
	go func() {
		lp := vkLongPollServer{}

		for {
			// Update Long Poll data

			if len(lp.Key) == 0 {
				response, err := v.Request("groups.getLongPollServer", url.Values{"group_id": {groupId}})
				if err != nil {
					log.Println("getLongPollServer error:", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}

				if len(lp.Ts) == 0 {
					lp.Ts = response.Get("ts").Str
				}
				lp.Server = response.Get("server").Str
				lp.Key = response.Get("key").Str

				log.Println("Long Poll server updated")
			}

			// Long Poll request

			resp1, err := http.Get(fmt.Sprintf("%s?act=a_check&key=%s&ts=%s&wait=25", lp.Server, lp.Key, lp.Ts))
			if err != nil {
				log.Println("LP poll error:", err.Error())
				time.Sleep(5 * time.Second)
				continue
			}

			body, err := ioutil.ReadAll(resp1.Body)
			resp1.Body.Close()

			if err != nil {
				log.Println("LP read error:", err.Error())
				continue
			}

			updates := gjson.GetBytes(body, "updates")

			// Long Poll error handling

			if !updates.Exists() {
				failedCode := gjson.GetBytes(body, "failed")
				if !failedCode.Exists() {
					log.Printf("LP json struct error: %#v\n", body)
					continue
				}

				switch failedCode.Int() {
				case 1: // history expired or lost
					lp.Ts = gjson.GetBytes(body, "ts").Str
				case 2: // key expired
					lp.Key = ""
				case 3: // data lost
					lp.Key = ""
					lp.Ts = ""
				default:
					log.Fatalf("LP unknown failed code: %s\n", failedCode.Str)
				}

				continue
			}

			updates.ForEach(func(_, value gjson.Result) bool {
				if value.Get("type").Str != "message_new" {
					return true
				}

				go event(v, value.Get("object"))
				return true
			})

			lp.Ts = gjson.GetBytes(body, "ts").Str
		}
	}()
}

type LongPollMessage struct {
	Date   int64 `json:"date"`
	FromId int64 `json:"from_id"`
	// Id             int           `json:"id"`
	// Out            int           `json:"out"`
	// PeerId         int64         `json:"peer_id"`
	Text string `json:"text"`
	// ConversationId int           `json:"conversation_message_id"`
	// FwdMessages    []interface{} `json:"fwd_messages"`
	// Important      bool          `json:"important"`
	// RandomId       int64         `json:"random_id"`
	// Attachments    []interface{} `json:"attachments"`
	Payload string `json:"payload"`
	// IsHidden       bool          `json:"is_hidden"`
}
