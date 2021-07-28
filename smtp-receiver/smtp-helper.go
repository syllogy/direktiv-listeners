package main

import (
	"regexp"
	"strings"
)

const sectionDelimiterRegex = `--[0-9]{1,100}-[0-9]{1,100}=:[0-9]{1,100}`

type SMTPPayload struct {
	Metadata    map[string]string
	Attachments []map[string]string
}

func handleSMTPPayload(payload string) (*SMTPPayload, error) {

	parsedPayload := new(SMTPPayload)
	parsedPayload.Metadata = make(map[string]string)
	parsedPayload.Attachments = make([]map[string]string, 0)

	var startIndex, endIndex int
	var inSection bool

	elems := strings.Split(payload, "\n")
	for k, str := range elems {

		if k < endIndex {
			continue
		}

		ok, err := regexp.Match(sectionDelimiterRegex, []byte(str))
		if err != nil {
			panic(err)
		}
		if ok {
			// this marks a new section, this is an attachment, or message body
			if !inSection {
				startIndex = k
				inSection = true
				continue
			} else {
				endIndex = k - 1
				section, err := handleSMTPSection(elems, startIndex, endIndex+1)
				if err != nil {
					panic(err)
				}

				startIndex = k - 1
				parsedPayload.Attachments = append(parsedPayload.Attachments, section)
			}
		}

		if !inSection {
			kv := strings.Split(str, ":")
			if len(kv) > 1 {
				k := kv[0]
				v := strings.Join(kv[1:], ":")

				parsedPayload.Metadata[k] = v
			}
			continue
		}

	}

	return parsedPayload, nil

}

func handleSMTPSection(elems []string, start, end int) (map[string]string, error) {

	sectionRange := elems[start:end]
	out := make(map[string]string)
	// var endOfHeaders bool
	// bodyLines := make([]string, 0)

	var endHeaders int

	for line, str := range sectionRange[1:] {

		if ok, _ := regexp.Match(sectionDelimiterRegex, []byte(str)); ok {
			continue
		}

		kv := strings.Split(str, ":")
		if len(kv) > 1 {
			k := kv[0]
			v := strings.Join(kv[1:], ":")

			out[k] = v
			continue
		}

		// if str == "" {
		// fmt.Println("THIS NEVER HAPPENS")
		// endOfHeaders = true
		endHeaders = line + 2
		break
		// }

		// if endOfHeaders {
		// 	bodyLines = append(bodyLines, str)
		// }

	}

	// fmt.Printf("ENDHEADERS: %d\n", endHeaders)

	out["PAYLOAD"] = strings.Join(sectionRange[endHeaders:], "\n")
	return out, nil
}
