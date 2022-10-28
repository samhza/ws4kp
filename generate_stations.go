//go:build exclude

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Station struct {
	ID        string `json:"StationId"`
	City      string `json:"City"`
	State     string `json:"State"`
	Latitude  string `json:"Latitude"`
	Longitude string `json:"Longitude"`
}

func main() {
	input, err := os.Open("stations.txt")
	if err != nil {
		log.Fatalln("Error opening stations.txt:", err)
	}
	output, err := os.Create("stations.js")
	if err != nil {
		log.Fatalln("Error creating stations.js:", err)
	}
	_, err = output.WriteString("var _StationInfo = {")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) != 83 {
			continue
		}
		if line[0] == '!' {
			continue
		}
		if isSpace(line) {
			continue
		}
		var (
			state     = line[0:2]
			city      = line[3 : 3+16]
			stationID = line[20 : 20+4]
			latitude  = line[39 : 39+7]
			longitude = line[47 : 47+7]
		)
		switch {
		case isSpace(stationID), isSpace(state):
			continue
		case stationID == "ICAO":
			// station ID header
			continue
		}
		station := Station{
			ID:        stationID,
			City:      normalizeCity(city),
			State:     state,
			Latitude:  degreesToDecimal(latitude),
			Longitude: degreesToDecimal(longitude),
		}
		var stationb []byte
		stationb, err = json.Marshal(station)
		if err != nil {
			break
		}
		if _, err = fmt.Fprintf(output, "%s:", stationID); err != nil {
			break
		}
		if _, err = output.Write(stationb); err != nil {
			break
		}
		if _, err = output.Write([]byte{','}); err != nil {
			break
		}
	}
	if _, err = output.Write([]byte{'}'}); err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("Error reading stations.txt:", err)
	}

}

func normalizeCity(s string) string {
	return strings.Title(strings.ToLower(strings.TrimSpace(s)))
}

func isSpace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func degreesToDecimal(degreesminutes string) string {
	// Example: 39 51N -> 39.85
	// Example: 104 39W -> -104.65

	split := strings.Split(degreesminutes, " ")
	degrees, err := strconv.ParseFloat(split[0], 32)
	if err != nil {
		log.Fatalln("Error parsing degrees:", err)
	}
	minutes, err := strconv.ParseFloat(split[1][:2], 32)
	if err != nil {
		log.Fatalln("Error parsing minutes:", err)
	}
	var sign float64 = 1
	switch split[1][2] {
	case 'S', 'W':
		sign = -1
	}
	return strconv.FormatFloat(sign*degrees+minutes/60, 'f', 2, 32)
}
