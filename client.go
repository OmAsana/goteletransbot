package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"

	//	"github.com/davecgh/go-spew/spew"
	"github.com/tubbebubbe/transmission"
	"github.com/tucnak/telebot"
	"gopkg.in/yaml.v2"
)

type botConfig struct {
	Telegram struct {
		Token string `yaml:"token"`
	}
	Transmission struct {
		Host string
		User string
		Pass string
	}
}

func (c *botConfig) Parse(data []byte) error {
	return yaml.Unmarshal(data, c)
}

func RegExpUrl(m telebot.Message) (url string) {
	//r, err := regexp.Compile(`^(https?:\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?$`)
	r, err := regexp.Compile(`^http(s)?`)
	if err != nil {
		fmt.Println("Smth wrong with regexp compile")
	}
	if r.MatchString(m.Text) == true {
		//	fmt.Println(m.Text)
		return m.Text
	}
	//fmt.Println(m.Text)
	return ""
}

func RegExpMagnet(m telebot.Message) (magnet string) {
	//r, err := regexp.Compile(`^magnet:\?xt=urn:(?:tree:tiger|[\w]+):([\w]+)[\S]+`)
	r, err := regexp.Compile(`^magnet:?`)
	if err != nil {
		fmt.Println("Smth wrong with regexp compile")
	}
	if r.MatchString(m.Text) == true {
		//	fmt.Println(m.Text)
		return m.Text
		fmt.Println("Magnet: " + m.Text)
	}
	fmt.Println("Not a magnet: " + m.Text)
	return ""
}

func DownloadTorrentUrl(url string, client transmission.TransmissionClient) (name string) {
	if url != "" {
		torrent := url
		cmd, _ := transmission.NewAddCmdByURL(torrent)
		torrentAdded, err := client.ExecuteAddCommand(cmd)
		if err != nil {
			fmt.Println(err)
		}
		//	spew.Dump(torrentAdded)
		//fmt.Println(torrentAdded.Name)
		return torrentAdded.Name
	}
	return ""
}

func DownloadMagnetLink(magnet string, client transmission.TransmissionClient) (name string) {
	if magnet != "" {
		cmd, _ := transmission.NewAddCmdByMagnet(magnet)
		torrentAdded, err := client.ExecuteAddCommand(cmd)
		if err != nil {
			fmt.Println(err)
		}
		//	spew.Dump(torrentAdded)
		//fmt.Println(torrentAdded.Name)
		return torrentAdded.Name
	}
	return ""
}

func TeleFileUrl(token string, m telebot.Message) (url2 string) {
	type GetFile struct {
		Ok     bool `json:"ok"`
		Result struct {
			File_id   string `json:"file_id"`
			File_size int    `json:"file_size"`
			File_path string `json:"file_path"`
		} `json:"result"`
	}

	JSON := new(GetFile)
	api := fmt.Sprintf("https://api.telegram.org/")
	getfile_url := api + fmt.Sprintf("bot%s/getFile?file_id=%s", token, m.Document.FileID)
	getJson(getfile_url, JSON)
	file_url := api + fmt.Sprintf("file/bot%s/%s", token, JSON.Result.File_path)
	return file_url
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	rep, err := ioutil.ReadAll(r.Body)

	err2 := json.Unmarshal(rep, &target)
	if err2 != nil {
		fmt.Println(err2)
	}
	return nil
}

func main() {
	var config botConfig
	data, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	Token := config.Telegram.Token
	Host := config.Transmission.Host
	User := config.Transmission.User
	Pass := config.Transmission.Pass
	// Telegram Bot
	bot, err := telebot.NewBot(Token)
	if err != nil {
		log.Fatalln(err)
	}

	// Transmission
	client := transmission.New(Host, User, Pass)

	messages := make(chan telebot.Message)
	bot.Listen(messages, 1*time.Second)
	for message := range messages {
		//		fmt.Println(message.Text)
		if TorrentUrl := RegExpUrl(message); TorrentUrl != "" {
			t_name := DownloadTorrentUrl(TorrentUrl, client)
			bot.SendMessage(message.Chat,
				"Added: "+t_name, nil)
		} else if MagnetLink := RegExpMagnet(message); MagnetLink != "" {
			t_name := DownloadMagnetLink(MagnetLink, client)
			bot.SendMessage(message.Chat,
				"Added: "+t_name, nil)
		}

		if message.Text == "/start" {
			bot.SendMessage(message.Chat,
				"no rexpt!", nil)
		}

		if message.Document.Exists() {
			if message.Document.Mime == "application/x-bittorrent" {
				TorrentUrl := TeleFileUrl(Token, message)
				t_name := DownloadTorrentUrl(TorrentUrl, client)
				bot.SendMessage(message.Chat,
					"Added: "+t_name, nil)
			}
		}

		if message.Text == "/hi" {
			bot.SendMessage(message.Chat,
				"Hello, "+message.Sender.FirstName+"!", nil)
		}
	}
}
