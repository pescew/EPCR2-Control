package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type config struct {
	Url          string `mapstructure:"URL"`
	User         string `mapstructure:"USER"`
	Pass         string `mapstructure:"PASS"`
	outletNumber int
	outletAction string
}

func main() {
	cfg, err := loadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	outletNum := flag.Int("outlet", -1, "outlet number to control. 0 for all outlets")
	outletAction := flag.String("action", "on", "action to perform (on, off, cycle)")
	flag.Parse()

	if *outletNum < 0 {
		log.Fatal("invalid outlet number")
	}

	cfg.outletNumber = *outletNum
	cfg.outletAction = *outletAction

	sessionCookie, err := submitLogin(cfg)
	if err != nil {
		log.Fatal("cannot login:", err)
	}

	err = outletToggle(cfg.Url, cfg.outletNumber, cfg.outletAction, sessionCookie)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("OK")
}

func outletToggle(url string, outletNumber int, toggleMode string, cookie string) error {
	mode := ""
	switch {
	case strings.ToLower(toggleMode) == "on":
		mode = "ON"
		if outletNumber > 0 {
			log.Printf("Turning ON Outlet #%d...", outletNumber)
		} else {
			log.Print("Turning ON ALL Outlets...")
		}
	case strings.ToLower(toggleMode) == "off":
		mode = "OFF"
		if outletNumber > 0 {
			log.Printf("Turning OFF Outlet #%d...", outletNumber)
		} else {
			log.Print("Turning OFF ALL Outlets...")
		}
	case strings.ToLower(toggleMode) == "cycle":
		mode = "CCL"
		if outletNumber > 0 {
			log.Printf("Cycling Outlet #%d...", outletNumber)
		} else {
			log.Print("Cycling ALL Outlets...")
		}
	}
	if mode == "" {
		return errors.New("outletToggle action not supported: " + toggleMode + ". Must be one of \"on\", \"off\", or \"cycle\"")
	}

	outletString := strconv.Itoa(outletNumber)
	if outletNumber == 0 {
		outletString = "a"
	}

	req, err := http.NewRequest("GET", strings.TrimRight(url, "/")+"/outlet?"+outletString+"="+mode, nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "DLILPC", Value: cookie})

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("outletToggle #" + outletString + " " + mode + " - status code not OK: " + strconv.Itoa(resp.StatusCode))
	}

	return nil
}

func submitLogin(cfg config) (sessionCookie string, err error) {
	pageBytes, err := loadPage(cfg.Url)
	if err != nil {
		return "", err
	}

	challenge, err := parseChallenge(string(pageBytes))
	if err != nil {
		return "", err
	}

	loginURL := strings.TrimRight(cfg.Url, "/") + "/login.tgi"

	resp, err := http.PostForm(loginURL, url.Values{"Username": {cfg.User}, "Password": {encodePass(cfg.User, cfg.Pass, challenge)}})
	if err != nil {
		return "", err
	}

	sessionCookie, found := GetStringBetweenStrings(resp.Header["Set-Cookie"][0], "DLILPC=\"", "\"")
	if !found {
		return "", errors.New("cookie not found")
	}

	return sessionCookie, nil
}

func loadPage(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	return content, nil
}

func encodePass(user, pass, challenge string) string {
	hasher := md5.New()
	hasher.Write([]byte(challenge + user + pass + challenge))
	return hex.EncodeToString(hasher.Sum(nil))
}

func parseChallenge(pageText string) (string, error) {
	challenge, found := GetStringBetweenStrings(pageText, "name=\"Challenge\" value=\"", "\">")
	if !found {
		return "", errors.New("challenge not found")
	}
	return challenge, nil
}

func loadConfig(path string) (config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return config{}, err
	}

	cfg := config{}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return config{}, err
	}

	uri, err := url.ParseRequestURI(cfg.Url)
	if err != nil {
		return config{}, err
	}
	cfg.Url = uri.String()

	return cfg, nil
}

func GetStringBetweenStrings(str string, startS string, endS string) (result string, found bool) {
	s := strings.Index(str, startS)
	if s == -1 {
		return result, false
	}
	newS := str[s+len(startS):]
	e := strings.Index(newS, endS)
	if e == -1 {
		return result, false
	}
	result = newS[:e]
	return result, true
}
