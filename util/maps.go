package util

// MergeMaps merges a list maps into the first one. B overrides A if keys collide.
func MergeMaps(base map[string]string, merges ...map[string]string) map[string]string {
	for _, m := range merges {
		for key, value := range m {
			base[key] = value
		}
	}
	return base
}

// IntersectMap will return map with the fields intersection from the 2 provided
// maps populated with the valueMap values.
func IntersectMap(templateMap, valueMap map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range templateMap {
		if innerTMap, ok := v.(map[string]interface{}); ok {
			if innerVMap, ok := valueMap[k].(map[string]interface{}); ok {
				result[k] = IntersectMap(innerTMap, innerVMap)
			}
		} else if innerTSlice, ok := v.([]interface{}); ok {
			if innerVSlice, ok := valueMap[k].([]interface{}); ok {
				items := []interface{}{}
				for idx, innerTSliceValue := range innerTSlice {
					if idx < len(innerVSlice) {
						if tSliceValueMap, ok := innerTSliceValue.(map[string]interface{}); ok {
							if vSliceValueMap, ok := innerVSlice[idx].(map[string]interface{}); ok {
								item := IntersectMap(tSliceValueMap, vSliceValueMap)
								items = append(items, item)
							}
						} else {
							items = append(items, innerVSlice[idx])
						}
					}
				}
				if len(items) > 0 {
					result[k] = items
				}
			}
		} else {
			if _, ok := valueMap[k]; ok {
				result[k] = valueMap[k]
			}
		}
	}
	return result
}
