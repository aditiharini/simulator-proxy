package querying

import (
	"encoding/json"

	config "github.com/aditiharini/simulator-proxy/config/experiment"
)

func ParseQuery(queryJson config.QueryJson) Query {
	if queryJson["type"] == "segment" {
		var segmentQuery SegmentQuery
		input := queryJson["input"].(config.QueryJson)
		segmentQuery.Input = ParseSingleOutputQuery(input)
		segmentQuery.NumSegments = int(queryJson["segments"].(float64))
		var outputs []string
		for _, output := range queryJson["output"].([]interface{}) {
			outputs = append(outputs, output.(string))
		}
		segmentQuery.Output = outputs
		return segmentQuery
	} else if queryJson["type"] == "range" {
		return ParseSingleOutputQuery(queryJson)
	} else if queryJson["type"] == "full_file" {
		return ParseSingleOutputQuery(queryJson)
	} else if queryJson["type"] == "stitch" {
		return ParseSingleOutputQuery(queryJson)
	} else if queryJson["type"] == "spotty" {
		return ParseSingleOutputQuery(queryJson)
	} else {
		panic("invalid query")
	}
}

func ParseSingleOutputQuery(queryJson config.QueryJson) SingleOutputQuery {
	jsonBytes, err := json.Marshal(queryJson)
	if err != nil {
		panic(err)
	}
	if queryJson["type"] == "range" {
		var rangeQuery RangeQuery
		input := queryJson["input"].(config.QueryJson)
		rangeQuery.Input = ParseSingleOutputQuery(input)
		rangeQuery.Length = int(queryJson["length"].(float64))
		rangeQuery.StartMilliOffset = int(queryJson["start"].(float64))
		rangeQuery.Output = queryJson["output"].(string)
		return rangeQuery
	} else if queryJson["type"] == "full_file" {
		var fullFileQuery FullFileQuery
		if err := json.Unmarshal(jsonBytes, &fullFileQuery); err != nil {
			panic(err)
		}
		return fullFileQuery
	} else if queryJson["type"] == "stitch" {
		var stitchQuery StitchQuery
		var queryInputs []Query
		inputs := queryJson["inputs"].([]config.QueryJson)
		for _, input := range inputs {
			queryInputs = append(queryInputs, ParseQuery(input))
		}
		stitchQuery.Inputs = queryInputs
		stitchQuery.Output = queryJson["output"].(string)
		return stitchQuery
	} else if queryJson["type"] == "spotty" {
		var spottyQuery SpottyQuery
		input := queryJson["input"].(config.QueryJson)
		spottyQuery.Input = ParseSingleOutputQuery(input)
		spottyQuery.Output = queryJson["output"].(string)
		spottyQuery.DisconnectThresholdLength = int(queryJson["disconnectThreshold"].(float64))
		spottyQuery.Length = int(queryJson["length"].(float64))
		spottyQuery.NumDisconnects = int(queryJson["disconnects"].(float64))
		return spottyQuery
	} else {
		panic("invalid query")
	}
}
