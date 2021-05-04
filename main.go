package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/ini.v1"
)

const defaultConfigFile = ".picoleafrc"

var verbose = flag.Bool("v", false, "Verbose")

// Client is a Nanoleaf REST API client.
type Client struct {
	Host  string
	Token string

	client http.Client
}

// Get performs a GET request.
func (c Client) Get(path string) string {
	if *verbose {
		fmt.Println("\nGET", path)
	}

	url := c.Endpoint(path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	if *verbose {
		fmt.Println("<===", string(body))
	}
	return string(body)
}

// Put performs a PUT request.
func (c Client) Put(path string, body []byte) {
	if *verbose {
		fmt.Println("PUT", path)
		fmt.Println("===>", string(body))
	}

	url := c.Endpoint(path)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	req.Body = ioutil.NopCloser(bytes.NewReader(body))

	res, err := c.client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}
}

// Endpoint returns the full URL for an API endpoint.
func (c Client) Endpoint(path string) string {
	return fmt.Sprintf("http://%s/api/v1/%s/%s", c.Host, c.Token, path)
}

// ListEffects returns an array of effect names.
func (c Client) ListEffects() ([]string, error) {
	body := c.Get("effects/effectsList")
	var list []string
	err := json.Unmarshal([]byte(body), &list)
	return list, err
}

// SelectEffect activates the specified effect.
func (c Client) SelectEffect(name string) error {
	req := EffectsSelectRequest{
		Select: name,
	}
	bytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	c.Put("effects/select", bytes)
	return nil
}

// BrightnessProperty represents the brightness of the Nanoleaf.
type BrightnessProperty struct {
	Value    int `json:"value"`
	Duration int `json:"duration,omitempty"`
}

// ColorTemperatureProperty represents the color temperature of the Nanoleaf.
type ColorTemperatureProperty struct {
	Value int `json:"value"`
}

// HueProperty represents the hue of the Nanoleaf.
type HueProperty struct {
	Value int `json:"value"`
}

// OnProperty represents the power state of the Nanoleaf.
type OnProperty struct {
	Value bool `json:"value"`
}

// SaturationProperty represents the saturation of the Nanoleaf.
type SaturationProperty struct {
	Value int `json:"value"`
}

// State represents a Nanoleaf state.
type State struct {
	On               *OnProperty               `json:"on,omitempty"`
	Brightness       *BrightnessProperty       `json:"brightness,omitempty"`
	ColorTemperature *ColorTemperatureProperty `json:"ct,omitempty"`
	Hue              *HueProperty              `json:"hue,omitempty"`
	Saturation       *SaturationProperty       `json:"sat,omitempty"`
}

// EffectsSelectRequest represents a JSON PUT body for `effects/select`.
type EffectsSelectRequest struct {
	Select string `json:"select"`
}

func main() {
	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		fmt.Printf("Failed to fetch current user: %v", err)
		os.Exit(1)
	}
	dir := usr.HomeDir
	configFilePath := filepath.Join(dir, defaultConfigFile)

	cfg, err := ini.Load(configFilePath)
	if err != nil {
		fmt.Printf("Failed to read file: %v", err)
		os.Exit(1)
	}

	client := Client{
		Host:  cfg.Section("").Key("host").String(),
		Token: cfg.Section("").Key("access_token").String(),
	}

	if *verbose {
		fmt.Printf("Host: %s\n\n", client.Host)
	}

	if flag.NArg() > 0 {
		cmd := flag.Arg(0)
		switch cmd {
		case "off":
			state := State{
				On: &OnProperty{false},
			}
			bytes, err := json.Marshal(state)
			if err != nil {
				fmt.Printf("Failed to marshal JSON: %v", err)
				os.Exit(1)
			}
			client.Put("state", bytes)
		case "on":
			state := State{
				On: &OnProperty{true},
			}
			bytes, err := json.Marshal(state)
			if err != nil {
				fmt.Printf("Failed to marshal JSON: %v", err)
				os.Exit(1)
			}
			client.Put("state", bytes)
		case "white":
			state := State{
				ColorTemperature: &ColorTemperatureProperty{6500},
			}
			bytes, err := json.Marshal(state)
			if err != nil {
				fmt.Printf("Failed to marshal JSON: %v", err)
				os.Exit(1)
			}
			client.Put("state", bytes)
		case "red":
			state := State{
				Brightness: &BrightnessProperty{60, 0},
				Hue:        &HueProperty{0},
				Saturation: &SaturationProperty{100},
			}
			bytes, err := json.Marshal(state)
			if err != nil {
				fmt.Printf("Failed to marshal JSON: %v", err)
				os.Exit(1)
			}
			client.Put("state", bytes)
		case "effect":
			doEffectCommand(client, flag.Args()[1:])
		}
	}
}

func doEffectCommand(client Client, args []string) {
	if len(args) < 1 {
		fmt.Println("usage: picoleaf effect list")
		fmt.Println("       picoleaf effect select <name>")
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "list":
		list, err := client.ListEffects()
		if err != nil {
			fmt.Printf("Failed retrieve effects list: %v", err)
			os.Exit(1)
		}
		for _, name := range list {
			fmt.Println(name)
		}
	case "select":
		if len(args) != 2 {
			fmt.Println("usage: picoleaf effect select <name>")
			os.Exit(1)
		}

		name := args[1]
		err := client.SelectEffect(name)
		if err != nil {
			fmt.Printf("Failed to select effect: %v", err)
			os.Exit(1)
		}
	}
}
