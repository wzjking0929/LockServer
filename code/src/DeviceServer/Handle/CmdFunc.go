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

	val, isExist = data["requestid"]
	if !isExist {
		log.Error("requestid 字段不存在:", data)
		return
	}
	requestid := val.(string)

	pushMsgDevCtrl(deviceID, requestid, 0, 1)
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

	if err := DBOpt.GetDataOpt().UpdateDeviceBarray(deviceID, battery); err != nil {
		log.Error("err:", err)
		return
	}

	pushMsgDevCtrl(deviceID, "", battery, 1)
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
	gwMap := make(map[string]interface{})
	deviceInfoArray := make([]Common.DeviceInfo, 0)
	gwMap["gw_mac"] = gatewayID

	dataMap := make(map[string]interface{})
	dataMap["cmd"] = "d2s_request_devices"
	dataMap["swm_gateway_info"] = gwMap
	dataMap["device_info"] = deviceInfoArray
	dataMap["statuscode"] = 0
	ackGateway(conn, dataMap)
	return

	//通过网关ID查询数据库,获取网关下的所有设备
	deviceList, err := DBOpt.GetDataOpt().GetDeviceIDList(gatewayID)
	if err != nil {
		log.Error("err:", err)
		return
	}
	log.Debug("deviceList:", deviceList)

	count := 0
	//设备列表过大，分包处理
	lenMap := len(deviceList)
	countDeviceList := 0

	if len(deviceList) == 0 {
		dataMap := make(map[string]interface{})
		dataMap["cmd"] = "d2s_request_devices"
		dataMap["swm_gateway_info"] = gwMap
		dataMap["device_info"] = deviceInfoArray
		dataMap["statuscode"] = 0
		ackGateway(conn, dataMap)
		return
	}
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

	val, isExist = deviceInfo["statuscode"]
	if !isExist {
		log.Error("statuscode  字段不存在:", data)
		return
	}
	statuscode := int(val.(float64))

	val, isExist = data["requestid"]
	if !isExist {
		log.Error("requestid 字段不存在:", data)
		return
	}
	requestid := val.(string)

	pushMsgSettingPassword(deviceID, ekeyValue, requestid, ekeyType, statuscode)
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

	val, isExist = deviceInfo["statuscode"]
	if !isExist {
		log.Error("statuscode  字段不存在:", data)
		return
	}
	statuscode := int(val.(float64))

	val, isExist = data["requestid"]
	if !isExist {
		log.Error("ekey_type 字段不存在:", data)
		return
	}
	requestid := val.(string)

	pushMsgCancelPassword(deviceID, ekeyValue, requestid, ekeyType, statuscode)
}

//刷卡开门上报
func cardOpenLockRecord(conn *gotcp.Conn, cmd string, deviceInfo map[string]interface{}) {
	val, isExist := deviceInfo["device_mac"]
	if !isExist {
		log.Error("device_mac 字段不存在:", deviceInfo)
		return
	}
	deviceID := val.(string)

	val, isExist = deviceInfo["openlock_cardnumber"]
	if !isExist {
		log.Error("openlock_cardnumber 字段不存在:", deviceInfo)
		return
	}
	ekeyValue := val.(string)

	val, isExist = deviceInfo["ekey_type"]
	if !isExist {
		log.Error("ekey_type 字段不存在:", deviceInfo)
		return
	}
	ekeyType := int(val.(float64))

	// val, isExist = deviceInfo["openlock_status"]
	// if !isExist {
	// 	log.Error("openlock_status 字段不存在:", deviceInfo)
	// 	return
	// }
	// openlockStatus := val.(string)
	val, isExist = deviceInfo["openlock_time"]
	if !isExist {
		log.Error("openlock_time 字段不存在:", deviceInfo)
		return
	}
	openlockTime := val.(string)

	val, isExist = deviceInfo["requestid"]
	if !isExist {
		log.Error("requestid 字段不存在:", deviceInfo)
		return
	}
	requestid := val.(string)

	dataMap := make(map[string]interface{})
	dataMap["cmd"] = "openlock_record_return"
	dataMap["device_mac"] = deviceID
	dataMap["requestid"] = requestid
	dataMap["ekey_type"] = ekeyType
	dataMap["openlock_time"] = openlockTime
	dataMap["statuscode"] = 0
	ackGateway(conn, dataMap)

	pushMsgCardOpenLockRsp(deviceID, ekeyValue, openlockTime, requestid, ekeyType)
}

////////////////////////////////////////////////////////////////////
//DevCtrl 控制开门
func DevCtrl(conn *gotcp.Conn, gatewayID, deviceID, requestid string) {
	dataMap := make(map[string]interface{})
	deviceInfo := make(map[string]interface{})
	deviceInfo["device_mac"] = deviceID
	deviceInfo["switchStatus"] = 1
	dataMap["cmd"] = "dev_ctrl"
	dataMap["requestid"] = requestid
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
func DevSettingPassword(conn *gotcp.Conn, devMac, keyValue, expireDate, requestid string, keyType int) {
	dataMap := make(map[string]interface{})
	dataMap["cmd"] = "dev_single_password_setting"
	dataMap["device_mac"] = devMac
	dataMap["requestid"] = requestid
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
func DevCancelPassword(conn *gotcp.Conn, devMac, keyValue, requestid string, keyType int) {
	dataMap := make(map[string]interface{})
	dataMap["cmd"] = "dev_single_password_cancel"
	dataMap["requestid"] = requestid
	dataMap["device_mac"] = devMac
	dataMap["ekey_type"] = keyType
	dataMap["ekey_value"] = keyValue
	dataMap["statuscode"] = 0

	ackGateway(conn, dataMap)
}


//@cmt DevResetEkeyInfo 清除节点卡号密码信息
/* e.g.
{  
	"cmd": "dev_reset",
	"requestid": "", 
	"device_info": {
		"device_mac": "123456789"
	}
	"statuscode ": 0
}
*/
func DevResetEkeyInfo(conn *gotcp.Conn, devMac, requestid string){

	dataMap:=make( map[string]interface{} )
	deviceInfo:=make( map[string]interface{} )

	dataMap["cmd"]="dev_reset"
	dataMap["requestid"]= requestid

	deviceInfo["device_mac"]= devMac
	dataMap["device_info"]= deviceInfo

	dataMap["statuscode"]= 0

	ackGateway(conn, dataMap)

}


//@cmt devResetRsp: 返回 *清楚节点卡号信息* 的结果给 应用层
func devResetRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}) {

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
	val, isExist = deviceInfo["reset_status"]
	if !isExist {
		log.Error("reset_status", data)
		return
	}
	resetStatus :=val.(float64)

	val, isExist = data["requestid"]
	if !isExist {
		log.Error("requestid字段不存在:", data)
		return
	}
	requestid := val.(string)

	pushMsgResetDev(deviceID, requestid, resetStatus)

}

//@cmt *设备常开常闭* 服务器-->网关
func DevNoncSet(conn *gotcp.Conn, devMac, requestid string, status int){
	dataMap:=make( map[string]interface{} )
	deviceInfo:=make( map[string]interface{} )

	dataMap["cmd"]="dev_nonc_set"
	dataMap["requestid"]= requestid

	deviceInfo["device_mac"]= devMac
	deviceInfo["status"]= status
	dataMap["device_info"]= deviceInfo

	ackGateway(conn, dataMap)	
}


//@cmt devNoncSet: *设备常开常闭*结果上传 (DeviceServer-->WechatAPI)
func devNoncSetRsp(conn *gotcp.Conn, cmd string, data map[string]interface{}){
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

	val, isExist = deviceInfo["status"]
	if !isExist {
		log.Error("status", data)
		return
	}
	status :=val.(int)
	val, isExist = deviceInfo["set_status"]
	if !isExist {
		log.Error("set_status", data)
		return
	}
	setStatus :=val.(int)

	val, isExist = data["requestid"]
	if !isExist {
		log.Error("requestid字段不存在:", data)
		return
	}
	requestid := val.(string)

	pushMsgDevNonc(deviceID, requestid, status, setStatus)
}


//@cmt 收到节点-->网关 的*节点注册*命令
func devBindGw(conn *gotcp.Conn, cmd string, data map[string]interface{}){
	val, isExist := data["device_info"]
	if !isExist {
		log.Error("device_info 字段不存在:", data)
		return
	}

	deviceInfo := val.(map[string]interface{}) //deviceinfo 
	val, isExist = deviceInfo["device_mac"]
	if !isExist {
		log.Error("device_mac 字段不存在:", data)
		return
	}
	deviceID := val.(string)   //device_mac

	val, isExist = data["requestid"]
	if !isExist {
		log.Error("requestid字段不存在:", data)
		return
	}
	requestid := val.(string)  //requestid

	val, isExist = data["gw_mac"]
	if !isExist {
		log.Error("gw_mac 字段不存在:", data)
		return
	}
	gwMac := val.(string)  //gw_mac	

	//@cmt 查询数据库判断是否 (gwMac, deviceID) 为绑定关系, 是则 status=1 否则 status=0
	isBind, err := DBOpt.GetDataOpt().IsGwDevBind(gwMac, deviceID)
	if err!=nil{
		log.Error("查询数据库错误 isGwDevBind：", gwMac, deviceID)
		return 
	}
	status:= 0
	if isBind==true{
		status=1
	}

	dataMap:=make( map[string]interface{} )
	deviceInfo2:=make( map[string]interface{} )

	dataMap["cmd"]="cmd_bind_gw"
	dataMap["gw_mac"]= gwMac
	dataMap["requestid"]= requestid
	deviceInfo2["device_mac"]= deviceID
	deviceInfo2["status"]= status  //status
	dataMap["device_info"]= deviceInfo2

	ackGateway(conn, dataMap) // server-->gw

}


//@cmt 设置节点测试模式
func DevSetTestMode(conn *gotcp.Conn, gwid, deviceMac string,txRate, txWait int, requestid string){
	dataMap:=make( map[string]interface{} )
	deviceInfo:=make( map[string]interface{} )

	dataMap["cmd"]="cmd_set_test_mode"
	dataMap["gw_mac"]= gwid
	dataMap["requestid"]= requestid
	deviceInfo["device_mac"]= deviceMac
	deviceInfo["tx_rate"]= txRate  
	deviceInfo["tx_wait"]= txWait 
	dataMap["device_info"]= deviceInfo

	ackGateway(conn, dataMap) // server-->gw
}


//@cmt set device normal mode
func DevSetWorkMode(conn *gotcp.Conn, gwid, deviceMac, requestid string){
	dataMap:=make( map[string]interface{} )
	deviceInfo:=make( map[string]interface{} )

	dataMap["cmd"]="cmd_set_work_mode"
	dataMap["gw_mac"]= gwid
	dataMap["requestid"]=requestid
	deviceInfo["device_mac"]= deviceMac
	dataMap["device_info"]= deviceInfo

	ackGateway(conn, dataMap) // server-->gw
}