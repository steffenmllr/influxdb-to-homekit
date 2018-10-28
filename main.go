package main

import (
	"encoding/json"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
	"github.com/influxdata/influxdb/client/v2"
)

const typeInfo = "14FA9D31-FC94-4F98-B00D-4AE878523748" // Parce Measurement Service

const (
	TypeTotalPower = "032B12CA-D4E8-4277-9021-188816FD00C6"
)

type InfoNumber struct {
	*characteristic.Int
}

type InfluxConfig struct {
	Addr     string
	Username string
	Password string
}

type AccessoriesConfig struct {
	Name        string
	Description string
	Unit        string
	Query       string
	Database    string
}

type HomekitConfig struct {
	Name         string
	Model        string
	Update       time.Duration
	Manufacturer string
	SerialNumber string
	Pin          string
	Port         string
	StoragePath  string `toml:"storage"`
}

type Config struct {
	Influx      InfluxConfig
	Accessories []AccessoriesConfig
	Homekit     HomekitConfig
}

type InfoService struct {
	*service.Service

	Name *characteristic.Name
	Info *InfoNumber
}

type Bridge struct {
	*accessory.Accessory
}

func NewBridge(c HomekitConfig) *Bridge {
	acc := Bridge{}
	info := accessory.Info{
		Name:         c.Name,
		Manufacturer: c.Manufacturer,
		Model:        c.Model,
	}
	acc.Accessory = accessory.New(info, accessory.TypeBridge)

	return &acc
}

func NewInfoNumber(val int) *InfoNumber {
	p := InfoNumber{characteristic.NewInt("")}
	p.Value = val
	p.Format = characteristic.FormatUInt64
	p.Perms = characteristic.PermsRead()

	return &p
}

func NewInfoService(name string, description string, unit string) *InfoService {
	nameChar := characteristic.NewName()
	nameChar.SetValue(name)

	info := NewInfoNumber(0)
	info.Type = TypeTotalPower
	info.Unit = unit
	info.Description = description
	svc := service.New(typeInfo)
	svc.AddCharacteristic(info.Characteristic)

	return &InfoService{svc, nameChar, info}
}

type Accessory struct {
	*accessory.Accessory

	Info   *InfoService
	Config AccessoriesConfig
}

func NewInfoAccessory(c AccessoriesConfig) *Accessory {
	info := accessory.Info{
		Name: c.Name,
	}
	a := accessory.New(info, accessory.TypeOther)
	svc := NewInfoService(c.Name, c.Description, c.Unit)
	a.AddService(svc.Service)

	return &Accessory{a, svc, c}
}

func getQueryValue(ic client.Client, c AccessoriesConfig) int {
	q := client.NewQuery(c.Query, c.Database, "ns")
	if response, err := ic.Query(q); err == nil && response.Error() == nil {
		newValue, _ := response.Results[0].Series[0].Values[0][1].(json.Number).Float64()
		return int(newValue)
	}

	return 0
}

func Update(iclient client.Client) {
	for _, acc := range allAccessories {
		currentValue := getQueryValue(iclient, acc.Config)
		acc.Info.Info.SetValue(currentValue)
	}
}

var config Config
var updateTicker *time.Ticker
var allAccessories []*Accessory

func main() {
	log.Debug.Enable()

	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Info.Panic(err)
	}

	iclient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.Influx.Addr,
		Username: config.Influx.Username,
		Password: config.Influx.Password,
	})

	defer iclient.Close()

	if err != nil {
		log.Info.Panic("Error creating InfluxDB Client: ", err.Error())
	}

	_, _, err = iclient.Ping(0)
	if err != nil {
		log.Info.Panic("Error pinging InfluxDB Cluster: ", err.Error())
	}

	for _, acc := range config.Accessories {
		infoAcc := NewInfoAccessory(acc)
		currentValue := getQueryValue(iclient, acc)
		infoAcc.Info.Info.SetValue(currentValue)
		allAccessories = append(allAccessories, infoAcc)
	}

	hkConfig := hc.Config{
		Pin:         config.Homekit.Pin,
		Port:        config.Homekit.Port,
		StoragePath: config.Homekit.StoragePath,
	}

	mainAccessories := make([]*accessory.Accessory, len(allAccessories))
	for i, acc := range allAccessories {
		mainAccessories[i] = acc.Accessory
	}
	t, err := hc.NewIPTransport(hkConfig, NewBridge(config.Homekit).Accessory, mainAccessories...)

	updateTicker = time.NewTicker(time.Minute * config.Homekit.Update)

	go func() {
		for _ = range updateTicker.C {
			Update(iclient)
		}
	}()

	if err != nil {
		log.Info.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
