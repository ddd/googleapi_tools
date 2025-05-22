package main

import (
	"encoding/json"
	"fmt"
)

const (
	size = 300
)

func genPayload(indices []int, dataType string) []byte {
	var result interface{}

	switch dataType {
	case "int":
		result = generateIntSlice(size)
	case "str":
		result = generateStrSlice(size)
	case "bool":
		result = generateBoolSlice(size)
	default:
		return nil
	}

	for i := len(indices) - 1; i >= 0; i-- {
		index := indices[i]
		newSlice := make([]interface{}, index)
		for j := 0; j < index-1; j++ {
			newSlice[j] = nil
		}
		newSlice[index-1] = result
		result = newSlice
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil
	}

	return payload
}

func generateIntSlice(n int) []int {
	slice := make([]int, n)
	for i := 0; i < n; i++ {
		slice[i] = i + 1
	}
	return slice
}

func generateStrSlice(n int) []string {
	slice := make([]string, n)
	for i := 0; i < n; i++ {
		slice[i] = fmt.Sprintf("x%d", i+1)
	}
	return slice
}

func generateBoolSlice(n int) []bool {
	slice := make([]bool, n)
	for i := 0; i < n; i++ {
		slice[i] = i%2 == 1
	}
	return slice
}
