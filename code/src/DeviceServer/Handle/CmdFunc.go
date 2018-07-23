package Handle

import (
	"DeviceServer/Common"
	"DeviceServer/Config"
	"DeviceServer/DBOpt"
	"gotcp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

//网关注册信息
func gatewayRegisterRsp(conn *gotcp.Conn, cmd string, dataMap map[string]interface{}) {
	val, isExist := dataMap["swm_gateway_info"]
	if !isExist {
		log.Error("swm_gateway_info 字段不存在:", dataMap)
		return
	}
	gwInfo := val.(map[string]interface{})
	val, isExist = gwInfo["gw_mac"]
	if !isExist {
		log.Error("gw_mac 字段不存在:", dataMap)
		return
	}
	gatewayID := val.(string)
	gatewayID = strings.ToUpper(gatewayID)

	ConnInfo[gatewayID] = conn

	conn.SetGatwayID(gatewayID)
	err := DBOpt.GetDataOpt().SetGatwayOnline(gatewayID)
	if err != nil {
		log.Error("err:", err)
	}

	//网关注册的时候，保存网关所注册的服务器地址到Redis
	err = Common.RedisServerOpt.Set(gatewayID, Config.GetConfig().HTTPServer, Config.GetConfig().RedisTimeOut)
	if err != nil {
		log.Error("err:", err)
		return
	}

	dataMap = make(map[string]interface{})
	dataMap["cmd"] = cmd
	dataMap["systemTime"] = time.Now().Format("2006-01-02 15:04:05")
	dataMap["statuscode"] = 0
	ackGateway(conn, dataMap)
}

//开门状态返回
func doorCtrlDealRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}) {
	val, isExist := data["device_info"]
	if !isExist {
		log.Error("device_info 字段不存在:", data)
		return
	}
	deviceInfo := val.(map[string]interface{})
	val, isExist = deviceInfo["device_mac"]
	if !isExist {
		log.Error("device_mac 字段不存在:", data)
		return
	}
	deviceID := val.(string)

	pushMsgDevCtrl(deviceID, -1, 1)
}

//电量信息上报
func doorReportBarryRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}) {
	val, isExist := data["device_info"]
	if !isExist {
		log.Error("device_info 字段不存在:", data)
		return
	}
	deviceInfo := val.(map[string]interface{})
	val, isExist = deviceInfo["device_mac"]
	if !isExist {
		log.Error("device_mac 字段不存在:", data)
		return
	}
	deviceID := val.(string)

	val, isExist = deviceInfo["battery"]
	if !isExist {
		log.Error("battery 字段不存在:", data)
		return
	}
	battery := val.(float64)

	pushMsgDevCtrl(deviceID, battery, 1)
}

//获取设备列表
func requestDeviceListRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}) {

	val, isExist := data["swm_gateway_info"]
	if !isExist {
		log.Error("swm_gateway_info 字段不存在:", data)
		return
	}

	gatewayInfo := val.(map[string]interface{})
	val, isExist = gatewayInfo["gw_mac"]
	if !isExist {
		log.Error("gw_mac 字段不存在:", data)
		return
	}
	gatewayID := val.(string)
	gatewayID = strings.ToUpper(gatewayID)
	time.Sleep(3 * time.Second)
	requestDeviceList2(conn, gatewayID)
}

func requestDeviceList2(conn *gotcp.Conn, gatewayID string) {
	//通过网关ID查询数据库,获取网关下的所有设备
	deviceList, err := DBOpt.GetDataOpt().GetDeviceIDList(gatewayID)
	if err != nil {
		log.Error("err:", err)
		return
	}
	log.Debug("deviceList:", deviceList)

	gwMap := make(map[string]interface{})
	deviceInfoArray := make([]Common.DeviceInfo, 0)
	gwMap["gw_mac"] = gatewayID

	count := 0
	//设备列表过大，分包处理
	lenMap := len(deviceList)
	countDeviceList := 0
	for k := range deviceList {
		countDeviceList++
		deviceInfo := new(Common.DeviceInfo)
		deviceInfo.DeviceID = k
		deviceInfo.RegStatus = 1
		deviceInfoArray = append(deviceInfoArray, *deviceInfo)
		//50个设备分包，或者最后一包
		if count == 50 || countDeviceList == lenMap {
			dataMap := make(map[string]interface{})
			dataMap["cmd"] = "d2s_request_devices"
			dataMap["swm_gateway_info"] = gwMap
			dataMap["device_info"] = deviceInfoArray
			dataMap["statuscode"] = 0
			ackGateway(conn, dataMap)

			count = 0
			deviceInfoArray = make([]Common.DeviceInfo, 0)
		}
	}
}

//下发卡号/密码响应
func devSettingPasswordRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}) {
	val, isExist := data["device_info"]
	if !isExist {
		log.Error("device_info 字段不存在:", data)
		return
	}
	deviceInfo := val.(map[string]interface{})
	val, isExist = deviceInfo["device_mac"]
	if !isExist {
		log.Error("device_mac 字段不存在:", data)
		return
	}
	deviceID := val.(string)

	val, isExist = deviceInfo["ekey_value"]
	if !isExist {
		log.Error("ekey_value 字段不存在:", data)
		return
	}
	ekeyValue := val.(string)

	val, isExist = deviceInfo["ekey_type"]
	if !isExist {
		log.Error("ekey_type 字段不存在:", data)
		return
	}
	ekeyType := int(val.(float64))

	val, isExist = deviceInfo["statuscode "]
	if !isExist {
		log.Error("statuscode  字段不存在:", data)
		return
	}
	statuscode := int(val.(float64))

	pushMsgSettingPassword(deviceID, ekeyValue, ekeyType, statuscode)
}

//取消下发卡号/密码响应
func devCancelPasswordRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}) {
	val, isExist := data["device_info"]
	if !isExist {
		log.Error("device_info 字段不存在:", data)
		return
	}
	deviceInfo := val.(map[string]interface{})
	val, isExist = deviceInfo["device_mac"]
	if !isExist {
		log.Error("device_mac 字段不存在:", data)
		return
	}
	deviceID := val.(string)

	val, isExist = deviceInfo["ekey_value"]
	if !isExist {
		log.Error("ekey_value 字段不存在:", data)
		return
	}
	ekeyValue := val.(string)

	val, isExist = deviceInfo["ekey_type"]
	if !isExist {
		log.Error("ekey_type 字段不存在:", data)
		return
	}
	ekeyType := int(val.(float64))

	val, isExist = deviceInfo["statuscode "]
	if !isExist {
		log.Error("statuscode  字段不存在:", data)
		return
	}
	statuscode := int(val.(float64))

	pushMsgCancelPassword(deviceID, ekeyValue, ekeyType, statuscode)
}

////////////////////////////////////////////////////////////////////
//devCtrl 控制开门
func devCtrl(conn *gotcp.Conn, gatewayID, deviceID string) {
	dataMap := make(map[string]interface{})
	deviceInfo := make(map[string]interface{})
	deviceInfo["device_mac"] = deviceID
	deviceInfo["switchStatus"] = 1
	dataMap["cmd"] = "dev_ctrl"
	dataMap["device_info"] = deviceInfo
	dataMap["statuscode"] = 0

	ackGateway(conn, dataMap)
}

//DevSettingPassword 发卡与设置开门密码
/*
 *参数说明： devMac 门锁ID
 *			keyValue 允许开门的卡号或者密码
 *			keyType 设备类型，0发卡，1密码
 *			expireDate 过期时间
 */
func DevSettingPassword(conn *gotcp.Conn, devMac, keyValue, expireDate string, keyType int) {
	dataMap := make(map[string]interface{})
	dataMap["cmd"] = "dev_single_password_setting"
	dataMap["dev_mac"] = devMac
	dataMap["ekey_type"] = keyType
	dataMap["ekey_value"] = keyValue
	dataMap["expiry_date"] = expireDate
	dataMap["statuscode"] = 0

	ackGateway(conn, dataMap)
}

//DevCancelPassword 取消卡号/密码开门
/*
 *参数说明： devMac 门锁ID
 *			keyValue 允许开门的卡号或者密码
 *			keyType 设备类型，0发卡，1密码
 */
func DevCancelPassword(conn *gotcp.Conn, devMac, keyValue string, keyType int) {
	dataMap := make(map[string]interface{})
	dataMap["cmd"] = "dev_single_password_cancel"
	dataMap["dev_mac"] = devMac
	dataMap["ekey_type"] = keyType
	dataMap["ekey_value"] = keyValue
	dataMap["statuscode"] = 0

	ackGateway(conn, dataMap)
}
