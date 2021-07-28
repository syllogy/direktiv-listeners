package main

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/mhale/smtpd"
	"gopkg.in/yaml.v2"
)

var gcConfig Config

type Config struct {
	SMTP struct {
		Address            string `yaml:"address"`
		Port               int    `yaml:"port"`
		InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	} `yaml:"smtp"`
	Direktiv struct {
		DirektivEndpoint string `yaml:"endpoint"`
		Token            string `yaml:"token"`
	} `yaml:"direktiv"`
	Internal string `json:"internal"`
}

type Attachment struct {
	Data        string `json:"data"`
	ContentType string `json:"content-type"`
	Name        string `json:"name"`
}

type ZipAttachment struct {
	Data string `json:"data"`
	Name string `json:"name"`
}

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	log.Println("reading message")
	// read message that was received
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		log.Printf("readmsg: %v\n", err)
		return err
	}
	// get subject header
	subject := msg.Header.Get("Subject")

	log.Println("reading body")
	// read data
	data, err = ioutil.ReadAll(msg.Body)
	if err != nil {
		log.Printf("readmsg body: %v\n", err)

		return err
	}
	pl, err := handleSMTPPayload(string(data))
	if err != nil {
		log.Printf("handleSMTPPayload: %v\n", err)

		return err
	}

	attachments := make([]Attachment, 0)
	for _, attachment := range pl.Attachments {
		var payload string
		var cte string
		var name string
		for k, v := range attachment {
			if k == "PAYLOAD" {
				payload = v
				continue
			}
			if k == "Content-Transfer-Encoding" {
				cte = v
			}
			if k == "Content-Type" {
				split := strings.Split(v, ";")[1]
				split = strings.TrimPrefix(split, " name=\"")
				split = strings.TrimSuffix(split, "\"\r")
				if strings.Contains(split, ".") {
					name = split
				} else {
					name = "body"
				}
			}
		}
		var attach Attachment
		attach.Data = payload
		attach.ContentType = cte
		attach.Name = name
		attachments = append(attachments, attach)
	}

	var message string
	for i, attach := range attachments {
		if strings.Contains(attach.ContentType, "base64") {
			sDec, _ := b64.StdEncoding.DecodeString(attach.Data)
			attachments[i].Data = string(sDec)
		}
		if attach.Name == "body" {
			message = attach.Data
		}
	}

	attachNew := make([]ZipAttachment, 0)
	for _, attach := range attachments {
		if attach.Name != "body" {
			attachNew = append(attachNew, ZipAttachment{
				Name: attach.Name,
				Data: attach.Data,
			})
		}
	}

	log.Println("creating cloud event")
	event := cloudevents.NewEvent()
	event.SetID("smtp-cloud")
	event.SetSource("smtp/msg")
	event.SetType("smtp")

	err = event.SetData(map[string]interface{}{
		"to":          to,
		"subject":     subject,
		"attachments": attachNew,
		"message":     message,
		"internal":    gcConfig.Internal,
	})
	if err != nil {
		log.Printf("ce set data: %v\n", err)

		return err
	}
	dd, err := json.Marshal(event)
	if err != nil {
		log.Printf("ce marshal: %v\n", err)

		return err
	}

	log.Println("sending cloud event")
	req, err := http.NewRequest("POST", gcConfig.Direktiv.DirektivEndpoint, bytes.NewReader(dd))
	if err != nil {
		log.Printf("send ce: %v\n", err)

		return err
	}

	// set access token if provided.
	if gcConfig.Direktiv.Token != "" {
		req.Header.Set("Authorization", gcConfig.Direktiv.Token)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: gcConfig.SMTP.InsecureSkipVerify},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("direktiv response: %v\n", err)

		return err
	}
	defer resp.Body.Close()

	return nil
}

func main() {
	if os.Args[1] == "" {
		log.Fatal("Config file needs to be provided ./smtp-receiver conf.yaml")
		return
	}

	// Open config file
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("Can not find file or we do not have permission to read")
		return
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&gcConfig); err != nil {
		log.Fatal("Failed decoding config file")
		return
	}

	log.Printf("%+v", gcConfig)
	log.Fatal(smtpd.ListenAndServe(fmt.Sprintf("%s:%v", gcConfig.SMTP.Address, gcConfig.SMTP.Port), mailHandler, "SMTP-Cloud-Direktiv", ""))
}
