package matchQuery

import (
	"bytes"
	"dictionary"
	"encoding/gob"
	"fmt"
	"index07"
	"sort"
	"time"
)

func MatchSearch(searchStr string, root *dictionary.TrieTreeNode, indexRoot *index07.IndexTreeNode, qmin int, qmax int) []index07.SeriesId {
	//划分查询串为VG
	start1 := time.Now().UnixMicro()
	var vgMap map[int]string
	vgMap = make(map[int]string)
	index07.VGConsBasicIndex(root, qmin, qmax, searchStr, vgMap)
	fmt.Println(vgMap)

	//查询每个gram对应倒排个数,并进行排序,把索引项放入sortGramInvertList
	var sortSumInvertList []SortKey
	var sortGramInvertList = make(map[string]index07.Inverted_index)
	for x := range vgMap {
		gram := vgMap[x]
		if gram != "" {
			invertIndex = nil
			var invertIndex2 index07.Inverted_index
			var invertIndex3 index07.Inverted_index
			SearchInvertedListFromCurrentNode(gram, indexRoot, 0)
			invertIndex2 = SearchInvertedListFromChildrensOfCurrentNode(indexNode, nil)
			if indexNode != nil && len(indexNode.AddrOffset()) > 0 {
				invertIndex3 = TurnAddr2InvertLists(indexNode.AddrOffset(), invertIndex3)
			}
			invertIndex = MergeMapsInvertLists(invertIndex2, invertIndex)
			invertIndex = MergeMapsInvertLists(invertIndex3, invertIndex)
			//去重 合并
			sortSumInvertList = append(sortSumInvertList, NewSortKey(len(invertIndex), gram))
			sortGramInvertList[gram] = invertIndex
		}
	}
	//fmt.Println(sortGramInvertList)
	//对sortSumInvertList中倒排表长度排序
	sort.SliceStable(sortSumInvertList, func(i, j int) bool {
		if sortSumInvertList[i].sizeOfInvertedList < sortSumInvertList[j].sizeOfInvertedList {
			return true
		}
		return false
	})
	end1 := time.Now().UnixMicro()

	var resArr []index07.SeriesId
	preSeaPosition := 0
	var preInverPositionDis []PosList
	var nowInverPositionDis []PosList
	start2 := time.Now().UnixMicro()
	for m := 0; m < len(sortSumInvertList); m++ {
		gramArr := sortSumInvertList[m].gram
		var nowSeaPosition int
		if gramArr != "" {
			for key := range vgMap {
				if vgMap[key] == gramArr {
					nowSeaPosition = key
				}
			}
			invertIndex = nil
			invertIndex = sortGramInvertList[gramArr]
			if invertIndex == nil {
				return nil
			}
			if m == 0 {
				for sid := range invertIndex {
					preInverPositionDis = append(preInverPositionDis, NewPosList(sid, make([]int, len(invertIndex[sid]), len(invertIndex[sid]))))
					nowInverPositionDis = append(nowInverPositionDis, NewPosList(sid, invertIndex[sid]))
					resArr = append(resArr, sid)
				}
			} else {
				for j := 0; j < len(resArr); j++ { //遍历之前合并好的resArr
					findFlag := false
					sid := resArr[j]
					if _, ok := invertIndex[sid]; ok {
						nowInverPositionDis[j] = NewPosList(sid, invertIndex[sid])
						for z1 := 0; z1 < len(preInverPositionDis[j].posArray); z1++ {
							z1Pos := preInverPositionDis[j].posArray[z1]
							for z2 := 0; z2 < len(nowInverPositionDis[j].posArray); z2++ {
								z2Pos := nowInverPositionDis[j].posArray[z2]
								if nowSeaPosition-preSeaPosition == z2Pos-z1Pos {
									findFlag = true
									break
								}
							}
							if findFlag == true {
								break
							}
						}
					}
					if findFlag == false { //没找到并且候选集的sid比resArr大，删除resArr[j]
						resArr = append(resArr[:j], resArr[j+1:]...)
						preInverPositionDis = append(preInverPositionDis[:j], preInverPositionDis[j+1:]...)
						nowInverPositionDis = append(nowInverPositionDis[:j], nowInverPositionDis[j+1:]...)
						j-- //删除后重新指向，防止丢失元素判断
					}
				}
			}
			preSeaPosition = nowSeaPosition
			copy(preInverPositionDis, nowInverPositionDis)
		}
	}
	end2 := time.Now().UnixMicro()
	fmt.Println("精确查询总花费时间（us）：", end2-start1)
	fmt.Println("精确查询划分查询串时间 + 查询索引树 + 排序gram对应倒排表list长度时间（us）：", end1-start1)
	fmt.Println("精确查询合并倒排时间（us）：", end2-start2)
	sort.SliceStable(resArr, func(i, j int) bool {
		if resArr[i].Id < resArr[j].Id && resArr[i].Time < resArr[j].Time {
			return true
		}
		return false
	})
	return resArr
}

var invertIndex index07.Inverted_index
var indexNode *index07.IndexTreeNode

func SearchInvertedListFromCurrentNode(gramArr string, indexRoot *index07.IndexTreeNode, i int) {
	if indexRoot == nil {
		return
	}
	if i < len(gramArr)-1 && indexRoot.Children()[gramArr[i]] != nil {
		SearchInvertedListFromCurrentNode(gramArr, indexRoot.Children()[gramArr[i]], i+1)
	}
	if i == len(gramArr)-1 && indexRoot.Children()[gramArr[i]] != nil {
		invertIndex = indexRoot.Children()[gramArr[i]].InvertedIndex()
		indexNode = indexRoot.Children()[gramArr[i]]
	}
}

func SearchInvertedListFromChildrensOfCurrentNode(indexNode *index07.IndexTreeNode, invertIndex2 index07.Inverted_index) index07.Inverted_index {
	if indexNode != nil {
		for _, child := range indexNode.Children() {
			if len(child.InvertedIndex()) > 0 {
				invertIndex2 = MergeMapsInvertLists(child.InvertedIndex(), invertIndex2)
			}
			if len(child.AddrOffset()) > 0 {
				var invertIndex3 = TurnAddr2InvertLists(child.AddrOffset(), nil)
				invertIndex2 = MergeMapsInvertLists(invertIndex3, invertIndex2)
			}
			invertIndex2 = SearchInvertedListFromChildrensOfCurrentNode(child, invertIndex2) //*
		}
	}
	return invertIndex2
}

func TurnAddr2InvertLists(addrOffset map[*index07.IndexTreeNode]int, invertIndex3 index07.Inverted_index) index07.Inverted_index {
	var res index07.Inverted_index
	for addr, offset := range addrOffset { //获取当前地址节点和他所有子节点的所有倒排
		invertIndex3 = nil
		//不需要addr的addrOffset吗？是的不需要？
		invertIndex3 = DeepCopy(addr.InvertedIndex())
		/*if addr != nil && len(addr.AddrOffset()) > 0 {
			index := TurnAddr2InvertLists(addr.AddrOffset(), nil)
			invertIndex3 = MergeMapsInvertLists(index, invertIndex3)
		}*/

		//invertIndex3 = SearchInvertedListFromChildrensOfCurrentNode(addr, invertIndex3)
		for _, list := range invertIndex3 {
			for i := 0; i < len(list); i++ {
				list[i] += offset
			}
		}
		res = MergeMapsInvertLists(invertIndex3, res)
	}
	return res
}

func MergeMapsInvertLists(map1 map[index07.SeriesId][]int, map2 map[index07.SeriesId][]int) map[index07.SeriesId][]int {
	if len(map2) > 0 {
		for sid1, list1 := range map1 {
			if list2, ok := map2[sid1]; !ok {
				map2[sid1] = list1
			} else {
				list2 = append(list2, list1...)
				list2 = UniqueArr(list2)
				sort.Ints(list2)
				map2[sid1] = list2
			}
		}
	} else {
		map2 = DeepCopy(map1)
	}
	return map2
}

func UniqueArr(m []int) []int {
	d := make([]int, 0)
	tempMap := make(map[int]bool, len(m))
	for _, v := range m { // 以值作为键名
		if tempMap[v] == false {
			tempMap[v] = true
			d = append(d, v)
		}
	}
	return d
}

func DeepCopy(src map[index07.SeriesId][]int) map[index07.SeriesId][]int {
	var dst map[index07.SeriesId][]int
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		fmt.Println(err)
	}
	gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(&dst)
	return dst
}

/*func Copy(value interface{}) interface{} {
	if valueMap, ok := value.(map[string]interface{}); ok {
		newMap := make(map[string]interface{})
		for k, v := range valueMap {
			newMap[k] = Copy(v)
		}

		return newMap
	} else if valueSlice, ok := value.([]interface{}); ok {
		newSlice := make([]interface{}, len(valueSlice))
		for k, v := range valueSlice {
			newSlice[k] = Copy(v)
		}

		return newSlice
	}
	return value
}*/