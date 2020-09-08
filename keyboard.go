package main

type VkKeyboard struct {
	OneTime bool         `json:"one_time"`
	Buttons [][]VkButton `json:"buttons"`
}

type VkButton struct {
	Action interface{} `json:"action"`
	Color  string      `json:"color"`
}

type VkTextButton struct {
	Type    string `json:"type"` // -> text
	Label   string `json:"label"`
	Payload string `json:"payload"`
}

type VkLocationButton struct {
	Type    string `json:"type"` // -> location
	Payload string `json:"payload"`
}

type VkPayButton struct {
	Type    string `json:"type"` // -> vkpay
	Payload string `json:"payload"`
	Hash    string `json:"hash"`
}

type VkAppsButton struct {
	Type    string `json:"type"` // -> open_app
	AppId   int    `json:"app_id"`
	OwnerId int    `json:"owner_id"`
	Payload string `json:"payload"`
	Label   string `json:"label"`
	Hash    string `json:"hash"`
}

func (_ *VkKeyboard) TxtBtn(text string, color string, payload string) (button VkButton) {
	return VkButton{
		Action: VkTextButton{
			Type:    "text",
			Label:   text,
			Payload: payload,
		},
		Color: color,
	}
}

func (v *VkKeyboard) AddRow(buttons ...VkButton) {
	v.Buttons = append(v.Buttons, buttons)
}
