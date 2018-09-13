package g

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/cihub/seelog"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"net/http"
	"time"
)

var (
	DLock sync.Mutex
	Root  string
	Db    *sql.DB
	Cfg   Config
	AuthipMap map[string]bool
)

func IsExist(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil || os.IsExist(err)
}

// Opening config file in JSON format
func ReadConfig(filename string) Config {
	config := Config{}
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		log.Fatal("Config Not Found!")
	} else {
		err = json.NewDecoder(file).Decode(&config)
		if err != nil {
			log.Fatal(err)
		}
	}
	return config
}

func GetRoot() string {
	//return "D:\\gopath\\src\\github.com\\gy-games\\smartping"
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal("Get Root Path Error:", err)
	}
	dirctory := strings.Replace(dir, "\\", "/", -1)
	runes := []rune(dirctory)
	l := 0 + strings.LastIndex(dirctory, "/")
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[0:l])
}

func ParseConfig(ver string) {
	Root = GetRoot()
	cfile := "config.json"
	if !IsExist(Root + "/conf/" + "config.json") {
		if !IsExist(Root + "/conf/" + "config-base.json") {
			log.Fatalln("[Fault]config file:", Root+"/conf/"+"config(-base).json", "both not existent.")
		}
		cfile = "config-base.json"
	}

	logger, err := seelog.LoggerFromConfigAsFile(Root + "/conf/" + "seelog.xml")
	seelog.ReplaceLogger(logger)
	Cfg = ReadConfig(Root + "/conf/" + cfile)
	if Cfg.Name == "" {
		Cfg.Name, _ = os.Hostname()
	}
	if Cfg.Ip == "" {
		Cfg.Ip = "127.0.0.1"
	}
	if Cfg.Mode == "" {
		Cfg.Mode = "local"
	}
	Cfg.Ver = ver
	if !IsExist(Root + "/db/" + "database.db") {
		if !IsExist(Root + "/db/" + "database-base.db") {
			log.Fatalln("[Fault]db file:", Root+"/db/"+"database(-base).db", "both not existent.")
		}
		src, err := os.Open(Root + "/db/" + "database-base.db")
		if err != nil {
			log.Fatalln("[Fault]db-base file open error.")
		}
		defer src.Close()
		dst, err := os.OpenFile(Root+"/db/"+"database.db", os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalln("[Fault]db-base file copy error.")
		}
		defer dst.Close()
		io.Copy(dst, src)
	}
	dbpath := Root + "/db/database.db"
	seelog.Info("Config loaded")
	Db, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatalln("[Fault]db open fail .", err)
	}
	for k, target := range Cfg.Targets {
		if target.Thdavgdelay == 0 {
			Cfg.Targets[k].Thdavgdelay = Cfg.Thdavgdelay
		}
		if target.Thdchecksec == 0 {
			Cfg.Targets[k].Thdchecksec = Cfg.Thdchecksec
		}
		if target.Thdloss == 0 {
			Cfg.Targets[k].Thdloss = Cfg.Thdloss
		}
		if target.Thdoccnum == 0 {
			Cfg.Targets[k].Thdoccnum = Cfg.Thdoccnum
		}
	}
	saveAuth()
}

func SaveCloudConfig(url string,flag bool) (Config,error){
	config := Config{}
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err!=nil{
		return config,err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(body,&config)
	if err != nil {
		return config,err
	}
	if flag == true{
		Cfg.Targets = config.Targets
		Cfg.Mode = "cloud"
		Cfg.Timeout = config.Timeout
		Cfg.Alertcycle = config.Alertcycle
		Cfg.Alerthistory = config.Alerthistory
		Cfg.Alertcycle = config.Alertcycle
		Cfg.Tsymbolsize = config.Tsymbolsize
		Cfg.Tline = config.Tline
		Cfg.Alertsound = config.Alertsound
		Cfg.Cendpoint = url
		Cfg.Authiplist = config.Authiplist
		saveAuth()
	}else{
		config.Mode = "cloud"
		config.Cendpoint = url
		config.Ip=Cfg.Ip
		config.Name=Cfg.Name
		config.Ver=Cfg.Ver
	}
	if err!=nil{
		return config,err
	}
	return config,nil
}

func SaveConfig() error {
	saveAuth()
	rrs, _ := json.Marshal(Cfg)
	var out bytes.Buffer
	errjson := json.Indent(&out, rrs, "", "\t")
	if errjson != nil {
		seelog.Error("[func:SaveConfig] Json Parse ", errjson)
		return errjson
	}
	err := ioutil.WriteFile(Root+"/conf/"+"config.json", []byte(out.String()), 0644)
	if err != nil {
		seelog.Error("[func:SaveConfig] Config File Write", err)
		return err
	}
	return nil
}

func saveAuth(){
	AuthipMap = map[string]bool{}
	Cfg.Authiplist = strings.Replace(Cfg.Authiplist, " ", "", -1)
	if Cfg.Authiplist!=""{
		authiplist := strings.Split(Cfg.Authiplist,",")
		for _, ip := range authiplist {
			AuthipMap[ip]=true
		}
	}
}
