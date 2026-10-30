package main

import "strings"

var deviceAreas = map[string]string{"CAB-001": "机房A区", "CAB-002": "机房A区", "CAB-003": "机房B区", "PIPE-001": "机房公共区"}

func validateLocation(deviceID, area string) (string, string, string) {
	deviceID = strings.ToUpper(strings.TrimSpace(deviceID))
	area = strings.TrimSpace(area)
	if deviceID != "" {
		expected, ok := deviceAreas[deviceID]
		if !ok {
			return "", "", "设备编号不存在"
		}
		if area != "" && area != expected {
			return "", "", "设备编号与所属区域不一致"
		}
		return deviceID, expected, ""
	}
	if area != "" && !valid(area, "机房A区", "机房B区", "机房公共区") {
		return "", "", "所属区域不正确"
	}
	return "", area, ""
}
