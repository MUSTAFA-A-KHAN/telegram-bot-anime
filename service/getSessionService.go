package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type CookieData struct {
	Cookie string `json:"cookie"`
}

func GetCookie() string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://raw.githubusercontent.com/MUSTAFA-A-KHAN/json-data-hub/refs/heads/main/cookie.json", nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", bodyText)
	var v CookieData
	err = json.Unmarshal(bodyText, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v.Cookie
}
