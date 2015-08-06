package main

import (
	"XPush"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/golang/protobuf/proto"
	"github.com/jinhao/apns"
	"os"
	"time"
)

func Do_push(raw_data []byte) error {
	msg := &XPush.CommonMsg{}
	err := proto.Unmarshal(raw_data, msg)
	if err != nil {
		log.Warnf("Do_push | proto Unmarshal failed, err:%s", err.Error())
		return err
	}

	ios_msg := msg.GetIosMsg()
	appid := ios_msg.GetAppid()
	log.Infof("Do_push |  appid:%s", appid)

	// 打包待发送消息
	m, err := pkg_apns_msg(msg)
	if err != nil {
		log.Warnf("Do_push | pkg_apns_msg err:%s", err.Error())
		return err
	}
	log.Infof("Do_push | msg:%s", m)

	// 推送消息到apns
	// 应用状态,生成环境为1，开发环境为2
	env := ios_msg.GetEnvironment()
	push_to_apns(appid, env, m)

	return nil
}

func pkg_apns_msg(msg *XPush.CommonMsg) (m apns.Notification, err error) {
	m = apns.NewNotification()
	p := apns.NewPayload()
	// 原始PayLoad
	raw_pay_load := msg.GetIosMsg().GetPayload()
	// 解析PayLoad
	var p_map map[string]interface{}
	if err := json.Unmarshal(raw_pay_load, &p_map); err != nil {
		log.Warnf("Do_push | json Unmarshal err:%s", err.Error())
		return m, err
	}

	// aps字段
	aps, ok := p_map["aps"]
	if !ok {
		log.Warn("pkg_apns_msg | aps not exist in p_map")
		//return m, errors.New("aps not exist in p_map")
	} else {
		aps_map, ok := aps.(map[string]interface{})
		// alert字段
		alert, ok := aps_map["alert"]
		if !ok {
			log.Warn("pkg_apns_msg | alert not exist in aps")
			return m, errors.New("alert not exist in aps")
		}
		alert_map, ok := alert.(map[string]interface{})
		p.APS.Alert.Body = alert_map["body"].(string)
		// badge字段
		badge, ok := aps_map["badge"]
		if !ok {
			log.Warn("pkg_apns_msg | badge not exist in aps")
		} else {
			badge_float, ok := badge.(float64)
			if !ok {
				log.Info("pkg_apns_msg | badge is not int")
			} else {
				badge_int := int(badge_float)
				p.APS.Badge = &badge_int
			}
		}
	}

	//token字段
	token := msg.GetDid()
	// token为64个字节
	if len(token) != 64 {
		log.Warnf("pkg_apns_msg | token:%s invalid", token)
		return m, errors.New(fmt.Sprintf("pkg_apns_msg | token:%s invalid", token))
	}
	m.DeviceToken = token

	//Expiration字段
	expire := msg.GetIosMsg().GetExpire()
	if expire > 0 {
		ex_time := time.Now()
		var duration time.Duration
		// 秒-> 纳秒(s -> ns)
		duration = time.Duration(expire * 1e9)
		ex_time.Add(duration)
		m.Expiration = &ex_time
	}

	// 自定义key-value对
	//customValues, ok := p_map
	for k, v := range p_map {
		if k == "aps" {
			continue
		}
		log.Infof("pkg_apns_msg | k:%s v:%s\n", k, v)
		err := p.SetCustomValue(k, v)
		if err != nil {
			log.Warnf("pkg_apns_msg | SetCustomValue err:%s", err.Error())
		}
	}
	m.Payload = p

	// 优先级
	priority := msg.GetIosMsg().GetPriority()
	if priority != apns.PriorityPowerConserve {
		m.Priority = apns.PriorityImmediate
	}
	i := 0
	m.Identifier = uint32(i)

	return m, nil
}

func push_to_apns(appid string, env int32, m apns.Notification) error {
	log.Info("push_to_apns enter")
	// 获取APPID对应的ios证书
	// TODO:先查看本地是否有，没有则从Mongodb获取，存本地
	certfile := "./cert/" + appid + ".pem"
	keyfile := certfile
	// 判断app状态: development or production
	// 生成环境为1，开发环境为2
	var apns_addr string
	if env == 1 {
		apns_addr = apns.ProductionGateway
	} else {
		env = 2
		apns_addr = apns.SandboxGateway
	}

	// 先查看该appid是否已经建立连接，没有则建立tls连接
	var conn *apns.Client
	if v, ok := Appid_client_map[appid]; ok {
		log.Infof("push_to_apns | conn for appid:%s exist", appid)
		conn = &v
	} else {
		log.Infof("push_to_apns | create new conn  for appid:%s", appid)
		c, err := apns.NewClientWithFiles(apns_addr, certfile, keyfile)
		if err != nil {
			log.Warnf("Could not create client", err.Error())
			return err
		}
		conn = &c
		//保存连接
		Appid_client_map[appid] = c
	}

	conn.Send(m)
	log.Info("push_to_apns | send msg success!")

	return nil
}

// 先判断本地是否存在指定的certfile，不存在则从MongoDB去获取
func certfile_exist(filename string, appid string, env int) (bool, error) {
	if filename == "" {
		return false, errors.New("filename is empty str")
	}
	_, err := os.Stat(filename)
	if err == nil {
		return true, nil
	} else {

	}
	return true, nil
}

//
func get_file_from_mongo(filename string, appid string, evn int) {

}
