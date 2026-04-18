package stringx

func ArrayRemoveElement(arr []string, target string) []string {
	var newArray []string
	for _, num := range arr {
		if num != target {
			newArray = append(newArray, num)
		}
	}
	return newArray
}

/**
 * @desc: 字符串数组去重
 * @param {[]uint} list
 * @return {*}
 */
func ArrayRemoveDup(list []string) []string {
	var x []string = []string{}
	for _, i := range list {
		if len(x) == 0 {
			x = append(x, i)
		} else {
			for k, v := range x {
				if i == v {
					break
				}
				if k == len(x)-1 {
					x = append(x, i)
				}
			}
		}
	}
	return x
}

/**
 * @desc: 查询字符串在不在数组里面
 * @param items 查询的数组
 * @param item 查询的字符串
 * @return {*}
 */
func StringInArray(item string, items []string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
