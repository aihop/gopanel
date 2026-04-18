package tencentcloud

import (
	"encoding/json"
	"fmt"
)

// dnspod 接口
func doDnsv3(secretId, secretKey, action string, reqMap map[string]interface{}) ([]byte, error) {
	payload, err := json.Marshal(reqMap)
	if err != nil {
		return nil, err
	}
	return doTencentCloudRequestv3(
		secretId,
		secretKey,
		"dnspod",
		"2021-03-23",
		action,
		"dnspod.tencentcloudapi.com",
		"",
		payload,
	)
}

func AddTxtRecord(secretId, secretKey, domain, rr, value string) error {
	reqMap := map[string]interface{}{
		"Domain":     domain,
		"SubDomain":  rr,
		"RecordType": "TXT",
		"RecordLine": "默认",
		"Value":      value,
	}
	_, err := doDnsv3(secretId, secretKey, "CreateRecord", reqMap)
	return err
}

func DeleteSubDomainRecords(secretId, secretKey, domain, rr string) error {
	// 首先需要查询对应的记录 ID
	reqMap := map[string]interface{}{
		"Domain":     domain,
		"SubDomain":  rr,
		"RecordType": "TXT",
	}
	bodyBytes, err := doDnsv3(secretId, secretKey, "DescribeRecordList", reqMap)
	if err != nil {
		return err
	}

	var result struct {
		Response struct {
			RecordList []struct {
				RecordId uint64 `json:"RecordId"`
			} `json:"RecordList"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return err
	}

	// 遍历删除
	for _, record := range result.Response.RecordList {
		delMap := map[string]interface{}{
			"Domain":   domain,
			"RecordId": record.RecordId,
		}
		_, err := doDnsv3(secretId, secretKey, "DeleteRecord", delMap)
		if err != nil {
			return fmt.Errorf("failed to delete record %d: %v", record.RecordId, err)
		}
	}
	return nil
}
