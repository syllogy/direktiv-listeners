package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
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
	ContentType string `json:"type"`
	Name        string `json:"name"`
}

type ZipAttachment struct {
	Data string `json:"data"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	log.Println("reading message")
	split := strings.Split(string(data), "\r\n")
	boundaryCheck := ""
	// Find the mime text boundary
	for _, x := range split {
		if strings.Contains(x, "Content-Type: multipart/mixed; boundary=") {
			y := strings.TrimPrefix(x, "Content-Type: multipart/mixed; boundary=\"")
			y = strings.TrimSuffix(y, "\"")
			boundaryCheck = y
			break
		}
	}

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

	pl, err := handleSMTPPayload(fmt.Sprintf("--%s", boundaryCheck), string(data))
	if err != nil {
		log.Printf("handleSMTPPayload: %v\n", err)

		return err
	}

	attachments := make([]Attachment, 0)

	ref, err := regexp.Compile(`filename="[\w-/.\ ()]*"`)
	if err != nil {
		log.Printf("regexp compiler: %v\n", err)
		return err

	}
	for _, attachment := range pl.Attachments {
		var payload string
		var cte string
		var name string
		for k, v := range attachment {
			if k == "PAYLOAD" {
				payload = v
				continue
			}

			if k == "Content-Disposition" {
				fmt.Println(v)
				d := ref.Find([]byte(v))
				if len(d) > 0 {
					f := strings.TrimPrefix(string(d), "filename=\"")
					f = strings.TrimSuffix(f, "\"")
					fmt.Println("filename: %s\n", f)
					name = f
				} else {
					name = "no-name"
				}
			}
			if k == "Content-Transfer-Encoding" {
				cte = v
			}
		}
		var attach Attachment
		attach.Data = payload
		attach.ContentType = cte
		attach.Name = "no-name"
		if name != "" {
			attach.Name = name
		}
		attachments = append(attachments, attach)
	}

	var message string
	for i, attach := range attachments {
		if strings.Contains(attach.ContentType, "base64") {
			// sDec, _ := b64.StdEncoding.DecodeString(attach.Data)
			// d := strings.TrimSpace(attachments[i].Data)
			// attachments[i].Data = fmt.Sprintf("%s", d)
			// fmt.Printf("D::%s\n", d)
		}
		fmt.Println(attach.Name)
		if attach.Name == "no-name" {
			message = attach.Data
		} else {
			attachments[i].Name = filepath.Base(attach.Name)
		}
	}

	attachNew := make([]ZipAttachment, 0)
	for _, attach := range attachments {
		if attach.Name != "no-name" {
			attachNew = append(attachNew, ZipAttachment{
				Name: attach.Name,
				Data: attach.Data,
				Type: strings.TrimSpace(strings.TrimSuffix(attach.ContentType, "\r")),
			})
		}
	}
	// fmt.Printf("%+v", attachNew)
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
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", gcConfig.Direktiv.Token)
		req.Header.Set("Direktiv-Token", true)
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

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("direktiv read response: %v\n", err)
		return err
	}

	log.Printf("Response from direktiv: %s\n", data)
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
