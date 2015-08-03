package main

import (
	"XPush"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jinhao/apns"
	"log"
)

var appid_client_map map[string]apns.Client

func Do_push(raw_data []byte) error {
	msg := &XPush.CommonMsg{}
	err := proto.Unmarshal(raw_data, msg)
	if err != nil {
		log.Printf("Do_push | proto Unmarshal failed, err:%s", err.Error())
	}
	dvc_token := msg.GetDid()
	ios_msg := msg.GetIosMsg()
	appid := ios_msg.GetAppid()
	log.Printf("Do_push | dvc_token:%s, appid:%s", dvc_token, appid)

	m, err := pkg_apns_msg(dvc_token)
	if err != nil {
		log.Fatal("Do_push | pkg_apns_msg err:%s", err.Error())
		return err
	}

	push_to_apns(appid, m)

	return nil
}

//func deserialize_data([])

func pkg_apns_msg(token string) (apns.Notification, error) {
	fmt.Print("Enter '<token> <badge> <msg>': ")

	var body string
	var badge int

	_, err := fmt.Scanf("%d %s", &badge, &body)
	if err != nil {
		log.Fatal("Something went wrong: %v\n", err.Error())
		//continue
	}
	p := apns.NewPayload()
	p.APS.Alert.Body = body
	p.APS.Badge = &badge

	p.SetCustomValue("link", "yourapp://precache/20140718")

	m := apns.NewNotification()
	m.Payload = p
	m.DeviceToken = token
	m.Priority = apns.PriorityImmediate
	i := 0
	m.Identifier = uint32(i)

	return m, nil
}

func push_to_apns(appid string, m apns.Notification) error {
	// 获取APPID对应的ios证书
	// 判断app状态: development or production

	// 先查看该appid是否已经建立连接，没有则建立tls连接
	var conn *apns.Client
	if v, ok := appid_client_map[appid]; ok {
		conn = &v
	} else {
		c, err := apns.NewClientWithFiles(apns.ProductionGateway, "apns.crt", "apns.key")
		if err != nil {
			log.Fatal("Could not create client", err.Error())
			return err
		}
		conn = &c
	}

	conn.Send(m)

	return nil
}
