package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "http://203.154.184.163:5011/api/authenticate/signIn"
	body := `{"username":"o.akrad","password":"akrad123123Zx!"}`

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.9,th;q=0.8")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", "http://203.154.184.163:5001")
	req.Header.Add("Referer", "http://203.154.184.163:5001/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36 Edg/145.0.0.0")
	req.Header.Add("X-TTT-PMRP", "ecffd46cf0f300f79f21afcac734ea9c")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)

	fmt.Printf("Status: %d\n", res.StatusCode)
	fmt.Printf("Body: %s\n", string(b))
}
