package main

import (
	//"encoding/json"
	"flag"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/samuel/go-zookeeper/zk"
	"goconfig"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"
)

var server_addr = flag.String("addr", "0.0.0.0:8002", "eventserver addr")
var zk_addr = flag.String("zkaddr", "127.0.0.1:2181,127.0.0.1:2182,127.0.0.1:2183", "zkserver addr")
var zk_path = flag.String("zkpath", "/xpush/iosproxy", "iosproxy report zk path")
var mgo_addr = flag.String("mgoaddr", "127.0.0.1:27017", "mongodb addr")
var db_xpush = flag.String("dbxpush", "xpush", "mongodb db name")
var collection_xpush = flag.String("collxpush", "push_history", "mongodb collection name")

func DealRequest(w http.ResponseWriter, r *http.Request) {
	log.Info("DealRequest")
	body, _ := ioutil.ReadAll(r.Body)
	go Do_push(body)
}

func read_cfg(cfg_file string) error {
	c, err := goconfig.ReadConfigFile(cfg_file)
	if err != nil {
		log.Infof("read_cfg | read_cfg | err:", err)
		return err
	}
	saddr, _ := c.GetString("server", "addr")
	z_addr, _ := c.GetString("zookeeper", "addr")
	z_path, _ := c.GetString("zookeeper", "path")
	m_addr, _ := c.GetString("mongodb", "addr")
	d_xpush, _ := c.GetString("mongodb", "db_xpush")
	c_xpush, _ := c.GetString("mongodb", "collection_xpush")

	if saddr != "" {
		server_addr = &saddr
	}
	if z_addr != "" {
		zk_addr = &z_addr
	}
	if z_path != "" {
		zk_path = &z_path
	}
	if m_addr != "" {
		mgo_addr = &m_addr
	}
	if d_xpush != "" {
		db_xpush = &d_xpush
	}
	if c_xpush != "" {
		collection_xpush = &c_xpush
	}
	return nil
}

/*
func parse_body(c *mgo.Collection, body []byte) {
	var push_data map[string]interface{}
	if err := json.Unmarshal(body, &push_data); err != nil {
		log.Error("parse_body | json Unmalshal err", err)
		panic(err)
	}
	if err := c.Insert(push_data); err != nil {
		log.Error("parse_body | collection Insert err", err)
	} else {
		log.Info("c insert success")
	}
}
*/

func zk_reginster(path string, c *zk.Conn) (string, error) {

	mgo_path := path + "/" + *server_addr
	//tPath, err := c.Create(mgo_path, []byte{}, 0, zk.WorldACL(zk.PermAll))
	tPath, err := c.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
	log.Infof("zk_reginster | path :%+v", mgo_path)
	tPath, err = c.Create(mgo_path, []byte{}, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		log.Warnf("zk_reginster | Create returned: %+v", err)
	}
	log.Infof("zk_reginster | create :%+v", tPath)
	return tPath, err
}

func zk_unreginster(path string, c *zk.Conn, exit chan os.Signal) {
	for sig := range exit {
		log.Warnf("zk_unreginster |  received ctrl+c(%v)\n", sig)
		err := c.Delete(path, -1)
		log.Infof("zk_unreginster | path :%+v", path)
		if err != nil {
			log.Warnf("zk_unreginster | Delete returned: %+v", err)
		}
		os.Exit(0)
	}

}

func deal_ctrl_c() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Printf("received ctrl+c(%v)\n", sig)
			os.Exit(0)
		}
	}()
}

func main() {
	runtime.GOMAXPROCS(5)
	//deal_ctrl_c()
	read_cfg("./ios_push.cfg")
	log_open()
	log.Warn("-----ios_push log------")

	flag.Parse()
	log.Warnf("main | server_addr:%s", *server_addr)
	log.Warnf("main | zk_addr:%s", *zk_addr)
	log.Warnf("main | mgo_addr:%s", *mgo_addr)
	log.Warnf("main | zk_path:%s", *zk_path)
	log.Warnf("main | db_xpush:%s", *db_xpush)
	log.Warnf("main | collection_xpush:%s", *collection_xpush)

	zks := strings.Split(*zk_addr, ",")
	fmt.Printf("zk_addr:%-v", zks)
	c, _, err := zk.Connect(zks, time.Second)
	if err != nil {
		panic(err)
		log.Errorf("zk_reginster | Connect zk:%s err:%s", zk_addr, err)
	}
	defer c.Close()

	tpath, err := zk_reginster(*zk_path, c)
	if err != nil {
		log.Errorf("main | zk_reginster err:%-v", err)
		return
	}
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)
	go zk_unreginster(tpath, c, exit)

	//pool = poolInit()
	http.HandleFunc("/push", DealRequest)
	http.HandleFunc("/hello/", DealRequest)
	err = http.ListenAndServe(*server_addr, nil)
	if err != nil {
		log.Errorf("main | ListenAndServer addr:%s err:%v", *server_addr, err)
	}
}
